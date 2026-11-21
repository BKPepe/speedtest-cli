package output

import (
	"slices"
	"testing"
)

func TestCleanupRunsWhatIsStillRegistered(t *testing.T) {
	// Cleanup is for a program that ends, so it leaves its mark for good.
	// Put the list back the way a new program finds it.
	t.Cleanup(func() {
		cleanupMu.Lock()
		cleanups, ending = nil, false
		cleanupMu.Unlock()
	})

	var ran []string
	note := func(name string) func() {
		return func() { ran = append(ran, name) }
	}

	OnExit(note("first"))
	forget := OnExit(note("done by its caller"))
	OnExit(note("last"))
	forget()

	Cleanup()
	if want := []string{"last", "first"}; !slices.Equal(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}

	// a second end of the program has nothing left to undo
	Cleanup()
	if len(ran) != 2 {
		t.Errorf("ran %v again", ran[2:])
	}

	// what starts while the program is ending is undone at once
	OnExit(note("started too late"))
	if want := []string{"last", "first", "started too late"}; !slices.Equal(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
}
