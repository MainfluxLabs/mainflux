# Pyscripts

Stores Python scripts, runs them on demand, and records what happened.

Scripts are group-scoped. Authorization is delegated to `things` via
`CanUserAccessGroup`: **viewer to read, editor to mutate and to run**. Running a
script executes code inside the service, so it is an edit-level action even
though it leaves the script unchanged.

## Writing a script

Plain Python. No entrypoint, no boilerplate - it runs top to bottom, exactly as
if you ran the file.

```python
readings = [r["v"] for r in message["payload"]["readings"]]
print("saw", len(readings), "readings")
sum(readings) / len(readings)
```

`message` is a global. `payload` is always present; `subtopic`, `created` and
`publisher_id` are supplied by the caller.

`print()` becomes logs, and so does anything written to `sys.stderr`. Errors come
back as a traceback starting at your line. If the last line is an expression its value is the
result, the same rule a notebook cell follows; end with a statement and there is
no value, and the run still succeeds. A value that is not JSON-serializable is
recorded as null. Raising is fine - the traceback comes back as the error,
starting at your line.

`numpy` and `pandas` are available. Nothing else is installable.

## Validation at save time

`POST /groups/:id/scripts` and `PUT /scripts/:id` reject source that exceeds
`MF_PYSCRIPTS_MAX_SCRIPT_SIZE` or does not compile, before anything is stored.
The syntax check runs `compile()` in a worker without forking, so a malformed
script is a `400` at the moment it is written rather than a failing run later.

Both checks live in the service, not only in the transport layer, so every
caller is bound by them.

## API

```
POST   /groups/:id/scripts      create
GET    /groups/:id/scripts      list by group
GET    /scripts/:id             view
PUT    /scripts/:id             update
PATCH  /scripts                 bulk delete
POST   /scripts/:id/run         run
GET    /scripts/:id/runs        run history
PATCH  /runs                    bulk delete
```

A script that raises **ran successfully**: `200` with `status: "fail"` and the
traceback in `error`. `5xx` is reserved for infrastructure failures - no worker
available, a corrupt frame.

## How a run executes

At startup the service launches `MF_PYSCRIPTS_WORKERS` Python processes, each
importing numpy and pandas once. Every run forks a throwaway child from a warm
worker: it inherits those modules copy-on-write, applies its limits, runs the
script, writes its result to an inherited pipe, and exits. About a millisecond,
against 300-800ms for a cold import.

The fork is what provides isolation - every run gets a pristine interpreter, so
a script can never affect the next one. `MF_PYSCRIPTS_WORKERS` is a concurrency
cap, not a safety control.

## Sandboxing

**This is trusted-operator grade, not hostile-multi-tenant.** Scripts are written
by authenticated group editors. The bar is that no run can harm anything outside
itself and every run is attributable - not that a determined attacker cannot
escape.

Nothing inside Python enforces this. `RestrictedPython` and AST allowlists fail
by construction once numpy and pandas are available: `np.ctypeslib` loads native
code and `pd.read_csv` reads files, both by design. Every control is at the OS
boundary.

Applied in the child, after the fork and before any user code:

| Control                                       | Stops                     |
|-----------------------------------------------|---------------------------|
| `RLIMIT_CPU`                                  | spin loops                |
| `RLIMIT_DATA`                                 | runaway allocation        |
| `RLIMIT_FSIZE`                                | writing files             |
| `RLIMIT_CORE`                                 | dumping memory to disk    |
| seccomp deny `socket`, `connect`, `bind`, ... | the network (see below)   |
| seccomp deny `setsid`, `setpgid`, `unshare`   | escaping the group kill   |
| seccomp deny `fork`, `clone`, `execve`, ...   | new processes (see below) |
| `/dev/null` on fds 0, 1 and 2                 | reaching Go's channels    |
| `setsid` + process-group kill                 | orphans outliving the run |

Applied in Go: `PR_SET_DUMPABLE(0)` at startup, a wall-clock deadline that
SIGKILLs the child's process group by the PID the worker reports, and an output
cap enforced while reading rather than after.

**`PR_SET_DUMPABLE(0)` is not optional.** Scripts run as unprivileged children of
the service and may read any file their uid allows - including
`/proc/1/environ`, which is the service's own environment. Without it a script
returns `MF_PYSCRIPTS_DB_PASS` as its result, and the scrubbed worker
environment buys nothing. Marking the service non-dumpable hands those /proc
entries to root so the children cannot read them. The service logs a warning if
the call fails; treat that as a deployment failure.

**Only the worker parent may declare a run finished.** The child keeps fd 3 in
order to emit logs and its result, and seccomp filters syscalls rather than file
descriptors, so user code can write arbitrary frames there - a script could
forge `{"kind":"exit"}`, have the pool mark the run complete and hand the worker
to the next caller while it was still running, desynchronising that worker's
stream and leaking one tenant's output into another's run. So `exit` and `pid`
travel on the control channel instead: the child has `/dev/null` on its fd 1 and
cannot reach it. Frames carry the job's sequence number, and anything on fd 3 is
treated as a claim, never as the end of a run.

**The child calls `setsid` and is killed by process group, and the filter then
forbids `setsid`/`setpgid`.** Anything the script forks shares that group, so it
dies with the run. The ordering matters: the child establishes its own session
*before* the filter loads, and a seccomp filter is inherited across `fork`, so
no descendant can move itself out of the group afterwards.

Without that denial the kill is escapable. A grandchild calling `os.setsid()`
lands in a different process group, survives both `kill(-pid)` and the parent's
`killpg`, and keeps the inherited data pipe - so it can write frames into later
runs on that worker, or simply take the worker down. Not theoretical: it killed
two workers during testing, and every request that then drew one failed.

**Every data frame names its job.** Each `log` and `result` frame carries the
job's sequence number, so output belonging to an earlier run is recognised and
dropped instead of being folded into whoever holds the worker next.

The stamp is not a security control - the child writes these frames, so it can
put any sequence number on them, and forging its own buys nothing it could not
already do by reporting whatever it likes for its own run. What it gives is
identity for stragglers, which is the case that actually crosses tenants.

A frame that does not match is therefore dropped and logged, and the worker is
kept. Retiring it would be the expensive answer to the one kind of residue that
is already handled, and would hand any caller a one-line way to empty the pool:
a single forged frame per run, each costing a respawn that blocks on importing
numpy. What still retires a worker is a state nobody can describe - a timeout, an
output-limit kill, or a drain that ran out of budget with frames still arriving.
Respawns are rate limited for the same reason, since exceeding the output cap is
also reachable from a script; the shortfall is left to `backfill`.

**A failure after the job has been handed over is not retried.** The retry
exists to mask a worker that was already dead, where the job never ran. Once the
frame has reached the worker the script may have run, so retrying would execute
it a second time, side effects included, and spend another worker doing it - a
script that kills its own worker could otherwise make one request run it three
times and retire three workers.

**A run is killed by pid, and the pid is held for the whole run.** The child has
its own session, so stopping its worker does not reach it. Any path that
abandons a run - a data channel the script broke itself with a bad frame header,
a worker that never confirms the kill, a caller that gave up during the
handshake - kills the child explicitly, and `stop` kills whatever run the worker
was hosting as a backstop. Without that a `time.sleep(300)` survived its own
failed request in its own session, one leaked process per call.

**A cache miss is reported by the worker parent, not by the run.** When a worker
no longer holds the source for a sha the pool believed it had, it says so on the
control channel and the pool resends the source. Keying that on the result error
instead - a string the child puts on fd 3 - let a cached script claim a miss and
have itself executed a second time on one request.

**A cancelled run is recorded as cancelled, not as a timeout.** A run context
carries the configured timeout and the caller's own cancellation on the same
channel. Reporting a client that hung up as `timeout after 10s` would invent a
limit the script never reached, and that error is what gets persisted, since runs
are saved on a context that outlives the request. The two are told apart by the
context's own error, giving `run cancelled by the caller` instead.

**Giving up on a job still disposes of its child.** The job frame reaches the
worker before the handshake that reports the forked pid, so abandoning a request
inside that window would leave a run executing with nothing holding its pid -
and because the child has its own session, retiring the worker does not reach it.
It would run until a limit caught it, or indefinitely if it were only sleeping.
Cancelling there waits briefly for the pid instead, kills it, and then discards
the worker. Verified by cancelling 60 requests against a `time.sleep(120)` script
at staggered sub-second deadlines: no process with `sid == pid` survived and
`pids.current` returned to its baseline.

**Do not run the container under an init shim.** Killing a run's group leaves
zombies owned by PID 1, so the service reaps them itself. Using `init: true` or
`--init` instead puts that shim at PID 1 with the same environment and without
`PR_SET_DUMPABLE`, so `/proc/1/environ` leaks the database password again - the
reaping and the secret protection have to live in the same process, and that
process is this service.

**The seccomp filter requires `pyseccomp`, which is installed in the service
image but is unavailable on platforms without seccomp, such as macOS.** Each
worker reports which sandbox it got in its ready frame, and the service logs a
warning at startup when the filter could not be installed:

```
python worker has no seccomp filter available: scripts are not prevented from using the network
```

Treat that warning as a deployment failure, not noise. Without the filter a
script can open sockets, and every service shares `mainfluxlabs-base-net` - so
it can reach `things`, `nats`, and other tenants' databases. Local development
against a host Python will always log it.

Applied by compose: `mem_limit`, `memswap_limit`, `cpus`, `pids_limit`,
`read_only`, `tmpfs`, non-root.

Two notes on why it is shaped this way. `RLIMIT_DATA` rather than `RLIMIT_AS`,
because numpy's BLAS backend reserves a large virtual address space at import
and an address-space cap makes `import numpy` itself raise `MemoryError`.
`RLIMIT_CPU` is CPU time, not wall clock, so a script blocked on a read never
trips it - the Go deadline is the only cover for that case.

### Known limitations of the sandbox

`multiprocessing` does not work: `Pool` needs to create semaphore files and
`RLIMIT_FSIZE=0` denies it (`OSError: [Errno 27] File too large`), which it hits
before the process-creation denial below would stop it anyway. Scripts should use
vectorised numpy, or threads, rather than process pools.

**A script cannot create a process.** The filter denies `fork`, `vfork` and
`execve`/`execveat` outright, and denies `clone` whenever `CLONE_THREAD` is clear
- so threads still work and only new processes are refused. `clone3` answers
`ENOSYS` rather than `EPERM` on purpose, because that is the error glibc retries
`clone` on; `EPERM` would make thread creation fail instead of falling back.

This is the control that bounds fork bombs, and it bounds them at zero: `for _ in
range(50): os.fork()` fails on the first call with `PermissionError`, and the
container's `pids.peak` stays at the pool's own 14. `subprocess` and
`os.posix_spawn` raise the same error; `os.system` returns status 127, glibc's
"could not create the child", since it reports failure through its return value
instead of raising.

Blocking process creation also closes a result-frame race. A forked child
inherits fd 3, so both copies of a forking script write a `result` frame and
whichever lands first wins the run - a script that forked once saw its own
`os.fork()` return `0`, meaning the frame that was recorded came from the child.

`RLIMIT_NPROC` is deliberately **not** used. It is a budget per uid across the
whole kernel rather than per process: every distroless service here runs as uid
65532, so its real headroom depends on what else is running, and at the default
of 64 it was already exhausted - `threading` failed with `can't start new thread`
and one run's forks could make `fork` fail for a concurrent run on a different
worker. An unpredictable limit that causes cross-run interference is worse than
none, and with `clone` filtered there is nothing left for it to do.

Note that `RLIMIT_DATA` does **not** bound forking, because each child is charged
its own copy of the limit. Before `clone` was filtered, a 400-fork bomb reached
`pids.peak` 137 while the container's own memory limit was never touched (`max 0`
in `memory.events`, peak 242MB of 1GB) - what it exhausted was the host's memory,
and the kernel's global OOM killer answered by culling processes across every
container on the machine, two warm workers among them. The cgroup `pids_limit`
remains the backstop for total process count, and is per-container rather than
per-uid, so it behaves the same everywhere.

A result that is not JSON-serializable - an `ndarray`, `np.int64`, a `DataFrame`,
`NaN` or `Infinity` - fails the run with `result cannot be returned: ...` rather
than being recorded as null. Call `.tolist()` or `float()` first.

### Not protected against

A kernel bug reachable through an allowed syscall. CPU side channels. Script
source is stored plaintext and readable by any group viewer, so it is not a
place for credentials, and run logs may carry payload data with the same
exposure. Hostile multi-tenancy would need gVisor or a microVM per run.

## Configuration

| Variable                        | Default     | Applies to                       |
|---------------------------------|-------------|----------------------------------|
| `MF_PYSCRIPTS_WORKERS`          | `2`         | pre-warmed pool size             |
| `MF_PYSCRIPTS_RUN_TIMEOUT`      | `10s`       | wall clock, Go-side              |
| `MF_PYSCRIPTS_ACQUIRE_TIMEOUT`  | `5s`        | wait for a free worker, then 503 |
| `MF_PYSCRIPTS_MAX_SCRIPT_SIZE`  | `65535`     | HTTP validation                  |
| `MF_PYSCRIPTS_MAX_PAYLOAD_SIZE` | `1048576`   | run request body                 |
| `MF_PYSCRIPTS_MAX_BODY_SIZE`    | `4194304`   | every other request body         |
| `MF_PYSCRIPTS_START_TIMEOUT`    | `60s`       | worker handshake at startup      |
| `MF_PYSCRIPTS_MAX_OUTPUT_SIZE`  | `1048576`   | capped pipe read                 |
| `MF_PYSCRIPTS_MAX_LOG_LINES`    | `256`       | log lines kept per run           |
| `MF_PYSCRIPTS_MAX_CPU_SECONDS`  | `5`         | `RLIMIT_CPU`                     |
| `MF_PYSCRIPTS_MAX_MEMORY`       | `268435456` | `RLIMIT_DATA`                    |
| `MF_PYSCRIPTS_MAX_FILE_SIZE`    | `0`         | `RLIMIT_FSIZE`                   |
| `MF_PYSCRIPTS_BLAS_THREADS`     | `1`         | `OMP_NUM_THREADS` et al.         |

One request is bounded by `ACQUIRE_TIMEOUT + RUN_TIMEOUT` once, not once per
retry, with the kill confirmation on top: the ceiling is 20s at the defaults.
Before that budget existed each attempt rebuilt both timeouts, so three attempts
could hold a caller for about 45s against a configured 10s, and nothing bounded
the request as a whole since the HTTP server sets no write deadline.

Pinning BLAS threads is the biggest single throughput factor and is invisible
until profiled: numpy defaults to one BLAS thread per visible core, so four
workers on a sixteen-core host is sixty-four threads contending over the
allocated `cpus`.

### Sizing the container

`MF_PYSCRIPTS_MEM_LIMIT` has to cover two things at once:

```
MEM_LIMIT  >=  WORKERS x ~120MB        numpy and pandas held resident per worker
             + WORKERS x MAX_MEMORY    worst case, every run sitting at its cap
```

At the defaults - 2 workers, a 256MB per-run cap - that is about 760MB, which is
why `MEM_LIMIT` is `1g`.

**It also has to stay below the memory the host actually gives Docker.** A limit
above that contains nothing. Instead of one container being killed, the kernel
runs a global OOM and takes whatever has the largest RSS - which is a
numpy-loaded worker, in this container or another service's. Docker Desktop
hands its VM far less than the host has; `docker info` prints the real figure.

This is not hypothetical. A `MEM_LIMIT` of `3g` against a 1.96GB Docker VM
produced `OOMKilled=true`, workers dying as `killed by killed` with no Python
traceback, and a `dmesg` showing the OOM killer taking a postgres in a different
container during the same episode. Scale `WORKERS` and `MAX_MEMORY` together
with the limit, and keep the total under what `docker info` reports.

## Known gaps

- **Run history grows without bound.** There is no retention policy yet.
- A worker that fails to start no longer stops the service: it boots on whatever
  came up and retries the shortfall every 30s, logging `started with N of M`.
  Only a total failure refuses to start. Treat a persistent shortfall as a sign
  the container is under-provisioned.
- Concurrency is capped per service, not per group, so one tenant can occupy
  the whole pool for up to `MF_PYSCRIPTS_ACQUIRE_TIMEOUT` at a time.
- A worker that dies while idle is detected on acquire and replaced, and the job
  retried on a fresh worker, so a corpse in the pool is no longer a failed
  request. Each discard logs a warning - a stream of them means scripts are
  killing workers.
- `/health` reports up before the worker pool has finished importing numpy.
- Shutdown does not yet drain in-flight runs or reap worker children.
- No pool-saturation or queue-wait metrics, so a saturated pool is not
  observable. Because runs shed rather than queue indefinitely, an overloaded
  service looks healthy while doing less work.
