// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package runner

// The python worker ships as source rather than a file: the runtime image copies
// exactly one artifact, and a .py file would have to be kept in sync with the
// binary it was built against. It is assembled from sections so each can be
// found without scrolling one long literal.
const workerSrc = pyPrelude + pyFraming + pySandbox + pyExec + pyLoop

// pyPrelude imports what the pool depends on and reports what it got. Both
// imports are optional in principle - numpy for the pre-warm that makes forking
// worthwhile, pyseccomp for the sandbox - so each is detected and named in the
// ready frame rather than failing silently.
//
// Both catch Exception, not ImportError, and that is the whole point. pyseccomp
// is a pure-python ctypes wrapper: it resolves libseccomp at import time and
// raises RuntimeError("Unable to find libseccomp") when the shared library is
// absent. Catching only ImportError let that escape, so instead of degrading to
// SANDBOX "none" the worker died mid-prelude - no ready frame, every spawn
// burning StartTimeout, and New reporting that no worker could be started. A
// missing optional dependency took the whole service down, three layers from
// its cause. The reason is folded into the reported string so the ready log
// names it rather than leaving a silent downgrade.
//
// fd 3 carries logs and results to Go. fd 4 is reserved for the Python API that
// will let a script call back into Mainflux, and is not read yet.
//
// hide_self is what actually protects the control channel. The service sets
// PR_SET_DUMPABLE on itself, but execve resets that flag, so the python process
// starts dumpable again - and a script could then reopen the worker's fd 1
// through /proc/<ppid>/fd/1 and write a forged exit frame, ending its own run in
// Go's eyes while continuing to run. Putting /dev/null on the child's fd 1 is
// not enough on its own. The flag is inherited across fork, so forked children
// are covered too, and the result is reported in the ready frame.
const pyPrelude = `
import os, sys, json, struct, ast, errno, signal, io

try:
    import numpy, pandas
    LOADED = "numpy " + numpy.__version__ + ", pandas " + pandas.__version__
except Exception as e:
    LOADED = "stdlib only (%s: %s)" % (type(e).__name__, e)

try:
    import pyseccomp
    SANDBOX = "seccomp"
except Exception as e:
    pyseccomp = None
    SANDBOX = "none (%s: %s)" % (type(e).__name__, e)

DATA, API = 3, 4
MAX_LINE = 8192

# MAX_RESULT keeps a result frame inside the reader's frame ceiling. Without it a
# script returning a few megabytes of string declared a length Go would not read,
# which left the stream with no safe place to resume and cost the worker - an
# infrastructure failure reported for what is really an oversized answer.
MAX_RESULT = 4 * 1024 * 1024
PR_SET_DUMPABLE, PR_GET_DUMPABLE = 4, 3

def hide_self():
    try:
        import ctypes
        libc = ctypes.CDLL(None, use_errno=True)
        libc.prctl(PR_SET_DUMPABLE, 0, 0, 0, 0)
        return libc.prctl(PR_GET_DUMPABLE, 0, 0, 0, 0) == 0
    except Exception:
        return False

HIDDEN = hide_self()
`

// pyFraming is the wire: length-prefixed JSON, plus the stdout/stderr capture
// that turns script output into log frames.
//
// send marshals with allow_nan=False because Go's json rejects the bare NaN that
// Python emits, which would make the frame undecodable and take the worker down
// with it. Out buffers rather than emitting a frame per write, because print()
// calls write() once per argument and a frame each would split every
// multi-argument print in half.
const pyFraming = `
def send(fd, obj):
    b = json.dumps(obj, allow_nan=False).encode()
    os.write(fd, struct.pack(">I", len(b)) + b)

SEQ = 0

def send_data(obj):
    obj["seq"] = SEQ
    send(DATA, obj)

def recv(fd):
    hdr = b""
    while len(hdr) < 4:
        c = os.read(fd, 4 - len(hdr))
        if not c:
            return None
        hdr += c
    n = struct.unpack(">I", hdr)[0]
    buf = b""
    while len(buf) < n:
        c = os.read(fd, n - len(buf))
        if not c:
            return None
        buf += c
    return json.loads(buf)

class Out:
    encoding = "utf-8"
    errors = "strict"
    closed = False
    def __init__(self):
        self.buf = ""
    def isatty(self):
        return False
    def readable(self):
        return False
    def seekable(self):
        return False
    def writable(self):
        return True
    def fileno(self):
        raise io.UnsupportedOperation("log stream has no descriptor")
    def writelines(self, lines):
        for line in lines:
            self.write(line)
    def close(self):
        self.flush()
    def emit(self, line):
        if len(line) > MAX_LINE:
            line = line[:MAX_LINE] + "... (truncated)"
        send_data({"kind": "log", "text": line})
    def write(self, s):
        self.buf += s
        while "\n" in self.buf:
            line, self.buf = self.buf.split("\n", 1)
            self.emit(line)
        if len(self.buf) > MAX_LINE:
            self.emit(self.buf)
            self.buf = ""
    def flush(self):
        if self.buf:
            self.emit(self.buf)
            self.buf = ""
`

// pySandbox holds every control applied to a run. All of it executes in the
// forked child: limiting the parent would cripple the process holding the numpy
// the pool exists to keep warm.
//
// The denied syscalls cover three things. Sockets, because scripts must not
// reach peer services on the shared docker network. setsid/setpgid/unshare,
// because isolate calls setsid before loading the filter so the child leads a
// killable group - a grandchild that moved itself out of that group would
// survive the kill and keep writing to the inherited data pipe.
//
// And process creation. A script that forks gets a second process holding the
// same data pipe, so both copies write a result frame and whichever lands first
// wins the run; forking in a loop also multiplies the memory cap, since each
// child is charged its own RLIMIT_DATA, and only the container's limit then
// bounds the total. clone is denied only when CLONE_THREAD is clear, so threads
// still work and only new processes are refused - clone3 answers ENOSYS rather
// than EPERM precisely so glibc falls back to clone instead of failing. execve
// is denied with them: it resets PR_SET_DUMPABLE, which is what keeps the
// control channel in hide_self out of reach.
//
// Two rlimit choices are deliberate. RLIMIT_CPU's soft limit sits below the hard
// one: with them equal the kernel goes straight to SIGKILL, indistinguishable
// from the deadline kill, whereas a second of headroom lets SIGXCPU arrive first
// so the cause can be named. And RLIMIT_DATA rather than RLIMIT_AS, because
// numpy's BLAS backend reserves a large address space at import and an AS cap
// makes "import numpy" itself raise MemoryError.
//
// Out has to pass for a text stream, not merely absorb writes. Libraries probe
// what they are printing to - isatty, fileno, encoding, writelines - and with
// only write and flush each probe raised AttributeError from inside the library,
// so a script using tqdm or rich failed with a traceback through someone else's
// code rather than its own. fileno raises io.UnsupportedOperation, which is what
// a stream without a descriptor is supposed to raise and what callers expect to
// catch.
//
// The seccomp filter covers the secondary ABIs as well as the native one. Rules
// apply only to architectures present in the filter, and anything reached
// through another ABI - i386 by int 0x80 under CONFIG_IA32_EMULATION, or x32 -
// would otherwise fall through to the ALLOW default and bypass every denial
// here. The arches are added before any rule, because libseccomp resolves a
// rule against the architectures the filter holds at the time it is added.
//
// isolate puts /dev/null on fds 0, 1 and 2: the child has no business reading job
// frames, fd 1 is Go's control channel, which a script must not be able to forge
// an exit frame on, and fd 2 is the worker's stderr pipe, which the service
// copies into its own log. Rebinding sys.stderr is not enough on its own, since
// os.write(2, ...) goes straight past it - and those bytes are never seen by the
// collector, so they count against neither the output cap nor the log limit.
const pySandbox = `
BLOCKED_SYSCALLS = (
    "socket", "socketpair", "connect", "bind", "listen",
    "accept", "accept4", "sendto", "sendmsg", "recvfrom", "recvmsg",
    "setsid", "setpgid", "unshare",
    "fork", "vfork", "execve", "execveat",
)

CLONE_THREAD = 0x00010000

def deny_new_processes(f):
    try:
        f.add_rule(pyseccomp.ERRNO(errno.EPERM), "clone",
                   pyseccomp.Arg(0, pyseccomp.MASKED_EQ, CLONE_THREAD, 0))
    except (ValueError, RuntimeError, OSError, AttributeError):
        pass
    try:
        f.add_rule(pyseccomp.ERRNO(errno.ENOSYS), "clone3")
    except (ValueError, RuntimeError, OSError):
        pass

SECONDARY_ARCHES = ("X86", "X32", "ARM")

def widen_arches(f):
    for name in SECONDARY_ARCHES:
        arch = getattr(pyseccomp.Arch, name, None)
        if arch is None:
            continue
        try:
            f.add_arch(arch)
        except Exception:
            pass

def deny_syscalls():
    if pyseccomp is None:
        return
    f = pyseccomp.SyscallFilter(pyseccomp.ALLOW)
    widen_arches(f)
    for name in BLOCKED_SYSCALLS:
        try:
            f.add_rule(pyseccomp.ERRNO(errno.EPERM), name)
        except (ValueError, RuntimeError, OSError):
            pass
    deny_new_processes(f)
    f.load()

def confine(lim):
    import resource

    cpu = lim.get("cpu_seconds")
    caps = [
        ("RLIMIT_DATA", lim.get("max_memory"), None),
        ("RLIMIT_FSIZE", lim.get("max_file_size"), None),
        ("RLIMIT_CORE", 0, None),
    ]
    if cpu is not None:
        caps.insert(0, ("RLIMIT_CPU", cpu, cpu + 1))

    for name, soft, hard in caps:
        if soft is None:
            continue
        what = getattr(resource, name, None)
        if what is None:
            continue
        try:
            resource.setrlimit(what, (soft, soft if hard is None else hard))
        except (ValueError, OSError):
            pass

    deny_syscalls()

def isolate(job):
    try:
        os.setsid()
    except OSError as e:
        send_data({"kind": "result", "value": None,
                    "error": "could not isolate the run: %s" % e})
        os._exit(1)
    devnull = os.open(os.devnull, os.O_RDWR)
    os.dup2(devnull, 0)
    os.dup2(devnull, 1)
    os.dup2(devnull, 2)
    os.close(devnull)
    sys.stdout = Out()
    sys.stderr = Out()
    confine(job.get("limits") or {})
`

// pyExec evaluates a script and compiles one for validation. Both run in a
// forked, confined child; doing either in the worker parent would leave it
// unbounded and one pathological input away from taking the worker down.
//
// run_code implements the result rule: the value is the last top-level
// expression, as a notebook cell behaves. It pops a trailing expression node,
// execs the rest and evals that node. The alternative - a magic "result"
// variable - fails silently on a typo.
//
// check_source catches BaseException because a pathological source can raise
// RecursionError or MemoryError, neither of which derives from Exception.
// user_traceback skips this file's frames so the author sees their own line.
const pyExec = `
def run_code(code, ns):
    tree = ast.parse(code, "script.py", "exec")
    last = None
    if tree.body and isinstance(tree.body[-1], ast.Expr):
        last = tree.body.pop()
    exec(compile(tree, "script.py", "exec"), ns)
    if last is None:
        return None
    return eval(compile(ast.Expression(last.value), "script.py", "eval"), ns)

def user_traceback(e):
    import traceback
    tb = e.__traceback__
    while tb and tb.tb_frame.f_code.co_filename != "script.py":
        tb = tb.tb_next
    return "".join(traceback.format_exception(type(e), e, tb)).strip()

def check_source(job):
    isolate(job)
    try:
        compile(job.get("code") or "", "script.py", "exec")
        send_data({"kind": "result", "value": None, "error": None})
    except SyntaxError as e:
        send_data({"kind": "result", "value": None, "error": str(e)})
    except BaseException as e:
        send_data({"kind": "result", "value": None,
                    "error": "%s: %s" % (type(e).__name__, e)})

def execute(job, code):
    isolate(job)
    try:
        ns = {"__name__": "__main__", "message": job["input"]}
        value = run_code(code, ns)
        sys.stdout.flush()
        sys.stderr.flush()
        try:
            encoded = json.dumps(value, allow_nan=False)
        except (TypeError, ValueError) as exc:
            send_data({"kind": "result", "value": None,
                        "error": "result cannot be returned: %s" % exc})
            return
        if len(encoded) > MAX_RESULT:
            send_data({"kind": "result", "value": None,
                        "error": "result cannot be returned: %d bytes exceeds the %d byte limit"
                                 % (len(encoded), MAX_RESULT)})
            return
        send_data({"kind": "result", "value": value, "error": None})
    except BaseException as e:
        sys.stdout.flush()
        sys.stderr.flush()
        send_data({"kind": "result", "value": None, "error": user_traceback(e)})
`

// pyLoop takes jobs forever. Every job is answered with a pid frame and an exit
// frame on the control channel, which the child cannot write to - so only this
// parent can declare a run over.
//
// fork_and_wait returns the pid immediately because Go has no handle on a child
// it did not create; without it a deadline could only kill the whole worker. It
// kills the process group after reaping, since the direct child is gone but
// anything it forked is not, and a survivor would write into the next run on
// this worker.
//
// A cache miss is reported rather than failed: Go caches independently and may
// believe this worker holds source it has evicted, and answers by resending it.
const pyLoop = `
MAX_CACHED = 64
cache = {}

def remember(sha, code):
    while len(cache) >= MAX_CACHED:
        cache.pop(next(iter(cache)))
    cache[sha] = code

def answered(seq, status=0, sig=0):
    send(1, {"kind": "exit", "seq": seq, "status": status, "signal": sig})

def fork_and_wait(seq, target):
    pid = os.fork()
    if pid == 0:
        target()
        os._exit(0)

    send(1, {"kind": "pid", "pid": pid, "seq": seq})
    _, status = os.waitpid(pid, 0)

    try:
        os.killpg(pid, signal.SIGKILL)
    except (ProcessLookupError, PermissionError, OSError):
        pass

    answered(seq,
             os.WEXITSTATUS(status) if os.WIFEXITED(status) else 0,
             os.WTERMSIG(status) if os.WIFSIGNALED(status) else 0)

send(1, {"kind": "ready", "loaded": LOADED, "sandbox": SANDBOX, "hidden": HIDDEN})

while True:
    job = recv(0)
    if job is None:
        break

    seq = job.get("seq", 0)
    SEQ = seq

    if job.get("check"):
        fork_and_wait(seq, lambda: check_source(job))
        continue

    if "code" in job:
        remember(job["checksum"], job["code"])

    code = cache.get(job["checksum"])
    if code is None:
        send(1, {"kind": "miss", "seq": seq})
        continue

    fork_and_wait(seq, lambda: execute(job, code))
`
