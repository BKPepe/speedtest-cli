//go:build unix

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// The CLI catches an interrupt to tidy the terminal up, and must then still
// end by that signal: a shell stops a loop around the CLI only when the CLI
// was killed by Ctrl+C, not when it exited on its own.
func TestInterruptedRunStillEndsByTheSignal(t *testing.T) {
	downloading := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/getIP.php":
			w.Write([]byte(`{"processedString":"127.0.0.1 - localhost","rawIspInfo":""}`))
		case "/garbage.php":
			select {
			case downloading <- struct{}{}:
			default:
			}
			chunk := make([]byte, 64<<10)
			for {
				if _, err := w.Write(chunk); err != nil {
					return
				}
			}
		}
	}))
	defer srv.Close()

	list := writeServerList(t, listedServer{1, "up", srv.URL + "/", "empty.php"})

	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			// the CLI would inherit that and, rightly, ignore the signal too
			if signal.Ignored(sig) {
				t.Skipf("the tests run with %v ignored", sig)
			}

			cmd := exec.Command(os.Args[0], "--local-json", list, "--server", "1", "--no-icmp", "--no-upload", "--duration", "30", "--json")
			cmd.Env = append(os.Environ(), runMainEnv+"=1")
			if err := cmd.Start(); err != nil {
				t.Fatalf("starting the CLI: %v", err)
			}
			exited := make(chan error, 1)
			go func() { exited <- cmd.Wait() }()

			select {
			case <-downloading:
			case err := <-exited:
				t.Fatalf("the CLI ended before its download: %v", err)
			case <-time.After(20 * time.Second):
				cmd.Process.Kill()
				t.Fatal("the download did not start")
			}

			cmd.Process.Signal(sig)

			select {
			case <-exited:
			case <-time.After(10 * time.Second):
				cmd.Process.Kill()
				t.Fatalf("the CLI kept running after %v", sig)
			}

			status := cmd.ProcessState.Sys().(syscall.WaitStatus)
			if !status.Signaled() || status.Signal() != sig {
				t.Errorf("ended with %v, want to be killed by %v", cmd.ProcessState, sig)
			}
		})
	}
}
