# GLock

*Project Status: ALPHA*

Runs a command if an associated lockfile is not acquired by another command.
Spiritual successor to `flock`.

Flock is a unix utility that allows you to specify a lockfile before running a command
so that only one instance of that command runs at a time.

A typical invocation of flock:
```sh
# this acquires the lockfile and runs the script
flock -xn /tmp/lockfile long_running_script.sh

# this fails immediately because another script has acquired the lockfile
flock -xn /tmp/lockfile long_running_script.sh
```

This makes it very convenient for controlling cron scripts that may run longer than their schedule.
For instance, a cron script may be scheduled to run every 30 mins but it's run time may end up
being 40 mins, longer than that 30 mins. This may be undesirable for scripts that require exclusive
access to some resource or scripts that when ran in parallel overutilize resources.

That being said, it is considered that engineering exclusive locks in the script itself
would be a better and more maintenable solution. However, there can be situations
that justify the use of `flock` and `glock` hopes to extend and improve the solutions.
Specifically, flock does not support the following uses cases:

1. Specifying a timeout for a script. A script may fail in such a way that it does not exit e.g. deadlocks.
   Flock doesn't allow you to specify that if the script doesn't exit in a specified amount of time, it is killed instead.
   You could potentially do the same with the `timeout` utility i.e. `timeout 5 flock ....` but this
   doesn't take the lockfile into consideration. For example in this case, once the script is killed,
   the lockfile needs to be released (deleted). Glock attempts to support this usecase.

2. Determining if a script owning a lockfile is dead. It is possible for flock to exit without
   releasing the lockfile. This could possibly be due to a *hard* exit e.g. signal-kill or OOM.
   In this scenario, because the lockfile was not removed, the next script will fail to start.
   Glock attempts to solve this by writing the pid of the process owning the lockfile *into* the
   lockfile. This allows the next invocation to query whether that pid is alive and if it's not,
   remove the *stale* lockfile and attempt to re-acquire a new lockfile.

Glock, however, does not currently support:

1. Shared locks also known as multiple readers, single writer locks.
2. Introspection tools to query the state of a running instance of glock (lockfile, its process).

## Installing

**Prebuilt binaries**:

1. Download a tarball from [Releases](https://github.com/kmwenja/glock/releases).
2. Extract the tarball: `tar -xvf glock-vX.Y.Z.tar.gz`. This will extract a directory called `glock`.
3. Copy the binary at `glock/glock` to a suitable path or run `glock/glock` directly.

**From Source**:
`go get -u -v github.com/kmwenja/glock`

## Usage

```sh
# help
glock

# run with defaults
glock echo hello world

# change the lockfile
glock -lockfile /tmp/mylockfile

# run with a specific timeout (10mins)
glock -timeout 600 echo hello world

# if another process has the lockfile,
# wait for them to be done for some time (20s) before quitting
glock -wait 20 echo hello world
```
