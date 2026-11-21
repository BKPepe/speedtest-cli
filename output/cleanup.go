package output

import "sync"

// Some of what the output does to the terminal outlives the code that did it:
// the progress spinner hides the cursor and shows it again only when it is
// stopped. A run that ends anywhere else, on an interrupt or in Fatal, would
// leave the terminal without a cursor. What has to be undone is therefore
// registered here and undone on those paths as well.

var (
	cleanupMu sync.Mutex
	cleanups  []*func()
	// ending is set once Cleanup has run: the program is on its way out
	ending bool
)

// OnExit registers fn to run if the program ends early. The returned function
// takes it off the list again, for the caller that got to do the work itself.
//
// Once the program is ending, fn runs at once. An interrupt does not stop the
// other goroutines, and what one of them starts in the moment before the
// program is gone would otherwise be left as it is.
func OnExit(fn func()) (forget func()) {
	entry := &fn

	cleanupMu.Lock()
	if ending {
		cleanupMu.Unlock()
		fn()
		return func() {}
	}
	cleanups = append(cleanups, entry)
	cleanupMu.Unlock()

	return func() {
		cleanupMu.Lock()
		defer cleanupMu.Unlock()
		for i, c := range cleanups {
			if c == entry {
				cleanups = append(cleanups[:i], cleanups[i+1:]...)
				return
			}
		}
	}
}

// Cleanup runs what OnExit registered, newest first, and forgets it. It is
// for a program that is about to end.
func Cleanup() {
	cleanupMu.Lock()
	pending := cleanups
	cleanups = nil
	ending = true
	cleanupMu.Unlock()

	for i := len(pending) - 1; i >= 0; i-- {
		(*pending[i])()
	}
}
