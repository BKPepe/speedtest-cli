//go:build !unix

package output

// CleanupOnInterrupt does nothing here. A program cannot send itself the
// signal again after catching it, so catching it would change the exit status
// an interrupted run ends with.
func CleanupOnInterrupt() {}
