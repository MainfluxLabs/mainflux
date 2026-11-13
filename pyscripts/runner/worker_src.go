// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package runner

// The python worker ships as source rather than a file: the runtime image copies
// exactly one artifact, and a .py file would have to be kept in sync with the
// binary it was built against. It is assembled from sections so each can be
// found without scrolling one long literal.
const workerSrc = pyPrelude + pyFraming + pySandbox + pyExec + pyLoop

// pyPrelude imports numpy and pyseccomp and reports what it got in the ready
// frame. Both imports catch Exception, because pyseccomp raises RuntimeError
// rather than ImportError when libseccomp is missing.
//
// fd 3 carries logs and results to Go. fd 4 is reserved for a future callback
// API and is not read yet.
//
// hide_self clears the dumpable flag, which execve resets, so a script cannot
// reach the worker's fd 1 through /proc and forge an exit frame. Forked
// children inherit it.
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

# MAX_RESULT keeps a result frame within the frame size Go will read.
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

// pyFraming is the wire: length-prefixed JSON, plus capture of script output as
// log frames. send uses allow_nan=False because Go rejects bare NaN. Out
// buffers by line because print() writes once per argument.
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

// pySandbox holds every control applied to a run, all in the forked child so
// the parent's warm numpy is unaffected.
//
// The seccomp filter is an allow-list with an EPERM default, so a call it
// misses surfaces as an OSError. On top of it: clone only with CLONE_THREAD, so
// no fork; clone3 and openat2 return ENOSYS so glibc falls back to calls the
// filter can inspect; open and openat are read-only and fail with EROFS;
// signals go only to the child's own pid; prlimit64 can only read. A syscall
// through another ABI hits the default and is denied.
//
// RLIMIT_CPU's soft limit sits below the hard one, so SIGXCPU names the cause
// before SIGKILL. RLIMIT_DATA is used rather than RLIMIT_AS, which breaks
// importing numpy. It does not charge anonymous MAP_SHARED mappings, so those
// are refused.
//
// Out implements enough of a text stream (isatty, fileno, encoding, writelines)
// for libraries such as tqdm and rich. isolate puts /dev/null on fds 0, 1 and
// 2, so a script can neither forge an exit frame on fd 1 nor bypass the output
// caps through fd 2.
const pySandbox = `
ALLOWED_SYSCALLS = (
    "exit", "exit_group", "rt_sigreturn", "restart_syscall",
    "set_tid_address", "set_robust_list", "get_robust_list", "rseq",
    "futex", "futex_waitv", "sched_yield",
    "sched_getaffinity", "sched_setaffinity",
    "sched_getparam", "sched_setparam",
    "sched_getscheduler", "sched_setscheduler",
    "sched_get_priority_max", "sched_get_priority_min", "sched_rr_get_interval",
    "getcpu", "membarrier", "nanosleep", "clock_nanosleep", "clock_nanosleep_time64",
    "getpid", "gettid", "getppid", "getuid", "geteuid", "getgid", "getegid",
    "getgroups", "getresuid", "getresgid", "getpgid", "getpgrp", "getsid",
    "getpriority", "uname", "sysinfo", "getrandom", "capget",
    "brk", "munmap", "mremap", "mprotect", "madvise",
    "mlock", "mlock2", "munlock", "mlockall", "munlockall", "msync", "mincore",
    "clock_gettime", "clock_gettime64", "clock_getres", "clock_getres_time64",
    "gettimeofday", "time", "times", "getrusage",
    "rt_sigaction", "rt_sigprocmask", "rt_sigpending",
    "rt_sigtimedwait", "rt_sigtimedwait_time64", "rt_sigsuspend",
    "sigaltstack", "signalfd", "signalfd4", "pause",
    "getrlimit", "ugetrlimit",
    "read", "readv", "preadv", "preadv2", "pread64",
    "write", "writev", "pwritev", "pwritev2", "pwrite64",
    "close", "close_range", "lseek", "_llseek",
    "dup", "dup2", "dup3", "fcntl", "fcntl64", "ioctl",
    "flock", "fsync", "fdatasync", "sync_file_range", "readahead",
    "fadvise64", "fadvise64_64",
    "stat", "stat64", "lstat", "lstat64", "fstat", "fstat64",
    "newfstatat", "fstatat64", "statx",
    "statfs", "statfs64", "fstatfs", "fstatfs64",
    "getdents", "getdents64", "readlink", "readlinkat", "getcwd",
    "access", "faccessat", "faccessat2", "umask",
    "poll", "ppoll", "ppoll_time64", "select", "_newselect",
    "pselect6", "pselect6_time64",
    "epoll_create", "epoll_create1", "epoll_ctl", "epoll_wait",
    "epoll_pwait", "epoll_pwait2",
    "eventfd", "eventfd2", "pipe", "pipe2",
    "timerfd_create", "timerfd_settime", "timerfd_gettime",
    "arch_prctl", "prctl", "set_thread_area", "get_thread_area",
)

SIGNAL_TO_SELF = ("kill", "tgkill", "rt_sigqueueinfo", "rt_tgsigqueueinfo")

CLONE_THREAD = 0x00010000

O_ACCMODE, O_WRONLY, O_RDWR, O_CREAT = 0o3, 0o1, 0o2, 0o100

MAP_SHARED, MAP_ANONYMOUS = 0x01, 0x20

# Denied by the default anyway; named so a write fails with EROFS rather than
# EPERM.
MUTATING_SYSCALLS = (
    "creat", "mkdir", "mkdirat", "rmdir",
    "link", "linkat", "symlink", "symlinkat",
    "rename", "renameat", "renameat2",
    "unlink", "unlinkat",
    "truncate", "ftruncate", "fallocate",
    "mknod", "mknodat",
    "chmod", "fchmod", "fchmodat",
    "chown", "fchown", "lchown", "fchownat",
    "utime", "utimes", "utimensat", "futimesat",
    "setxattr", "lsetxattr", "fsetxattr",
    "removexattr", "lremovexattr", "fremovexattr",
)

def allow_threads(f):
    try:
        f.add_rule(pyseccomp.ALLOW, "clone",
                   pyseccomp.Arg(0, pyseccomp.MASKED_EQ, CLONE_THREAD, CLONE_THREAD))
    except (ValueError, RuntimeError, OSError, AttributeError):
        pass
    try:
        f.add_rule(pyseccomp.ERRNO(errno.ENOSYS), "clone3")
    except (ValueError, RuntimeError, OSError):
        pass

def allow_signals_to_self(f, own):
    for name in SIGNAL_TO_SELF:
        try:
            f.add_rule(pyseccomp.ALLOW, name, pyseccomp.Arg(0, pyseccomp.EQ, own))
        except (ValueError, RuntimeError, OSError, AttributeError):
            pass

def allow_reads_deny_writes(f):
    for name, flags in (("open", 1), ("openat", 2)):
        try:
            f.add_rule(pyseccomp.ALLOW, name,
                       pyseccomp.Arg(flags, pyseccomp.MASKED_EQ, O_ACCMODE, 0))
        except (ValueError, RuntimeError, OSError, AttributeError):
            pass
        for mask, val in ((O_ACCMODE, O_WRONLY), (O_ACCMODE, O_RDWR), (O_CREAT, O_CREAT)):
            try:
                f.add_rule(pyseccomp.ERRNO(errno.EROFS), name,
                           pyseccomp.Arg(flags, pyseccomp.MASKED_EQ, mask, val))
            except (ValueError, RuntimeError, OSError, AttributeError):
                pass
    try:
        f.add_rule(pyseccomp.ERRNO(errno.ENOSYS), "openat2")
    except (ValueError, RuntimeError, OSError):
        pass
    for name in MUTATING_SYSCALLS:
        try:
            f.add_rule(pyseccomp.ERRNO(errno.EROFS), name)
        except (ValueError, RuntimeError, OSError):
            pass

def allow_read_rlimit(f):
    try:
        f.add_rule(pyseccomp.ALLOW, "prlimit64", pyseccomp.Arg(2, pyseccomp.EQ, 0))
    except (ValueError, RuntimeError, OSError, AttributeError):
        pass

def allow_mmap_except_shared_anon(f):
    # mmap is allowed except for anonymous MAP_SHARED, which RLIMIT_DATA does
    # not charge. ENOMEM names the refusal.
    for name in ("mmap", "mmap2"):
        try:
            f.add_rule(pyseccomp.ALLOW, name, pyseccomp.Arg(3, pyseccomp.MASKED_EQ, MAP_ANONYMOUS, 0))
        except (ValueError, RuntimeError, OSError, AttributeError):
            pass
        try:
            f.add_rule(pyseccomp.ALLOW, name, pyseccomp.Arg(3, pyseccomp.MASKED_EQ, MAP_SHARED, 0))
        except (ValueError, RuntimeError, OSError, AttributeError):
            pass
        try:
            f.add_rule(pyseccomp.ERRNO(errno.ENOMEM), name,
                       pyseccomp.Arg(3, pyseccomp.MASKED_EQ,
                                     MAP_SHARED | MAP_ANONYMOUS, MAP_SHARED | MAP_ANONYMOUS))
        except (ValueError, RuntimeError, OSError, AttributeError):
            pass

def install_filter():
    if pyseccomp is None:
        return
    f = pyseccomp.SyscallFilter(pyseccomp.ERRNO(errno.EPERM))
    for name in ALLOWED_SYSCALLS:
        try:
            f.add_rule(pyseccomp.ALLOW, name)
        except (ValueError, RuntimeError, OSError):
            pass
    allow_threads(f)
    allow_signals_to_self(f, os.getpid())
    allow_reads_deny_writes(f)
    allow_read_rlimit(f)
    allow_mmap_except_shared_anon(f)
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

    install_filter()

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

// pyExec evaluates a script and compiles one for validation, both in a forked,
// confined child. run_code returns the last top-level expression, as a notebook
// cell does. check_source catches BaseException because a pathological source
// can raise RecursionError or MemoryError. user_traceback skips this file's
// frames.
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

// pyLoop takes jobs forever, answering each with a pid frame and an exit frame
// on the control channel, which only this parent can write. fork_and_wait
// reports the pid at once so Go can kill a run at its deadline, and kills the
// process group after reaping so nothing a run forked survives into the next.
// The source travels with every job rather than being cached, so one run cannot
// read another's.
const pyLoop = `
def answered(seq, status, sig, nonce):
    send(1, {"kind": "exit", "seq": seq, "status": status, "signal": sig, "nonce": nonce})

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

    # The nonce is minted after the child is dead, so a script cannot emit it.
    # Go sees its echo on the data channel only if the stream is frame-aligned;
    # a partial frame swallows it and the worker is retired.
    nonce = os.urandom(16).hex()
    send(DATA, {"kind": "flush", "seq": seq, "nonce": nonce})
    answered(seq,
             os.WEXITSTATUS(status) if os.WIFEXITED(status) else 0,
             os.WTERMSIG(status) if os.WIFSIGNALED(status) else 0,
             nonce)

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

    fork_and_wait(seq, lambda: execute(job, job.get("code") or ""))
`
