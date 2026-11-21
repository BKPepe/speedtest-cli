//go:build unix

package output

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// cleanupGrace is how long an interrupted program waits for its cleanup.
const cleanupGrace = time.Second

// CleanupOnInterrupt makes an interrupt or a request to terminate run Cleanup
// before it ends the program.
func CleanupOnInterrupt() {
	// A signal the program was started to ignore stays ignored. Catching it
	// would run the cleanup for a program that then carries on.
	var caught []os.Signal
	for _, sig := range []os.Signal{syscall.SIGINT, syscall.SIGTERM} {
		if !signal.Ignored(sig) {
			caught = append(caught, sig)
		}
	}
	if len(caught) == 0 {
		return
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, caught...)

	go func() {
		sig := <-sigs

		// The cleanup writes to the terminal, and a terminal that is stuck
		// blocks it. The signal has to end the program all the same, so the
		// cleanup gets a moment and no more.
		done := make(chan struct{})
		go func() {
			Cleanup()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(cleanupGrace):
		}

		// Stop catching the signal and send it again, so that the program
		// ends the way it would have without this handler. A shell or a
		// script running it then still sees an interrupted run and not a
		// finished one.
		signal.Reset(caught...)
		if syscall.Kill(syscall.Getpid(), sig.(syscall.Signal)) == nil {
			// the signal ends the process, do not exit ahead of it
			select {}
		}
		os.Exit(1)
	}()
}
