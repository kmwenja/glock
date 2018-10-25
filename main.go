package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

var VERSION = ""

func main() {
	var (
		timeout  = flag.Int("timeout", 60, "number of seconds to wait for the command to terminate, otherwise force terminate. Use -1 to indicate 'wait forever'")
		lockfile = flag.String("lockfile", "/tmp/glockfile", "file to acquire to ensure the command can be run. If file exists, quit.")
		wait     = flag.Int("wait", 10, "number of seconds to wait to acquire the lockfile, otherwise quit. Use -1 to indicate 'wait and retry every 10s forever'")
		version  = flag.Bool("version", false, "print version")
	)
	flag.Parse()

	if *version {
		fmt.Printf("Version: %s\n", VERSION)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf("Usage: glock [options] command arg1 arg2 arg3 ....\n\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if !glock(*lockfile, *wait, *timeout, args) {
		os.Exit(1)
	}
}

func glock(lockfile string, wait int, timeout int, command []string) bool {
	// try acquiring the lock file
	log("obtaining lockfile: %s", lockfile)
	start := time.Now()
	for {
		err := lockFile(lockfile)
		if err == nil {
			defer func() {
				err = os.Remove(lockfile)
				if err != nil {
					logErr(errors.Wrap(err, "could not remove lockfile"))
				}
				log("released lockfile: %s", lockfile)
			}()
			break
		}

		logErr(errors.Wrap(err, "lock file error:"))

		// if we can't obtain lockfile, wait as instructed
		if wait > -1 {
			if time.Since(start) >= time.Duration(wait)*time.Second {
				// we waited long enough, quitting
				logErr(fmt.Errorf("could not obtain lockfile after waiting %ds", wait))
				return false
			}
		}

		logErr(fmt.Errorf("waiting 1s to try again"))
		time.Sleep(1 * time.Second)
	}
	log("obtained lockfile: %s", lockfile)

	// run command, and start timing
	// if command does not exit before timer, quit
	cmdString := strings.Join(command[0:], " ")
	log("running command (timeout: %ds): %s", timeout, cmdString)

	cmd := command[0]
	args := command[1:]
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		logErr(errors.Wrap(err, "could not start command"))
		return false
	}

	if timeout == -1 {
		// wait forever
		if err := c.Wait(); err != nil {
			logErr(errors.Wrap(err, "command exited with an error"))
			return false
		}
		log("successfully ran command")
		return true
	}

	timeoutDur := time.Duration(timeout) * time.Second
	done := make(chan error, 1)
	go func() {
		done <- c.Wait()
	}()
	select {
	case <-time.After(timeoutDur):
		if err := c.Process.Kill(); err != nil {
			logErr(errors.Wrap(err, "could not kill command"))
			return false
		}
		logErr(fmt.Errorf("command took longer than timeout and was killed"))
		return false
	case err := <-done:
		if err != nil {
			logErr(errors.Wrap(err, "command exited with an error"))
			return false
		}
		log("successfully ran command")
		return true
	}
}

func lockFile(filename string) error {
	// check if file exists first
	err := checkExisting(filename)
	if err != nil {
		return errors.Wrap(err, "check existing error")
	}

	f, err := os.OpenFile(
		filename,
		os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.Wrap(err, "could not create lockfile")
	}
	defer f.Close()

	// write pid into lockfile so that the owner can be traced
	_, err = fmt.Fprintf(f, "%d\n", os.Getpid())
	if err != nil {
		return errors.Wrap(err, "could not write pid to lockfile")
	}

	return nil
}

func checkExisting(filename string) error {
	f, err := os.OpenFile(
		filename, os.O_RDONLY, 0600)
	if err != nil {
		pe, ok := err.(*os.PathError)
		if !ok {
			return errors.Wrap(err, "unknown file error")
		}

		if pe.Err.Error() != "no such file or directory" {
			return errors.Wrap(err, "could not open file")
		}

		// lockfile does not exist, this is fine
		return nil
	}
	defer f.Close()

	// lockfile exists, get pid of owner
	var pid int
	_, err = fmt.Fscanf(f, "%d\n", &pid)
	if err != nil {
		// TODO potentially remove invalid lockfiles
		return errors.Wrap(err, "could not read from existing lockfile")
	}

	// check if owner is still alive
	process, err := os.FindProcess(pid)
	if err != nil {
		return errors.Wrapf(err, "failed while finding process %d", pid)
	}
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return fmt.Errorf("lockfile in use by another process")
	}
	if err.Error() != "os: process already finished" {
		return errors.Wrapf(err, "failed while finding process: %d", pid)
	}

	// owner of pid already finished so remove lockfile
	// TODO do this after closing the file
	err = os.Remove(filename)
	if err != nil {
		return errors.Wrapf(err, "could not remove existing lockfile")
	}

	return nil
}

func log(s string, args ...interface{}) {
	newS := fmt.Sprintf(s, args...)
	fmt.Fprintf(os.Stdout, "glock: %s\n", newS)
}

func logErr(e error) {
	fmt.Fprintf(os.Stderr, "glock: %v\n", e)
}
