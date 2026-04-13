package flux

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// Reloader manages a hot-reload lifecycle for a command. It monitors the
// filesystem for changes to .go files and restarts a child process.
type Reloader struct {
	Args     []string
	Interval time.Duration
}

// NewReloader creates a new Reloader for the given command and arguments.
func NewReloader(args ...string) *Reloader {
	return &Reloader{
		Args:     args,
		Interval: 500 * time.Millisecond,
	}
}

// Run starts the file watcher and the managed subprocess. It blocks until
// a termination signal (SIGINT, SIGTERM) is received.
func (r *Reloader) Run() error {
	fmt.Printf("%s   🔄  Hot Reload Active (Watching for changes...)\n%s", colorBrightCyan, colorReset)

	if len(r.Args) == 0 {
		return fmt.Errorf("no command provided")
	}

	var child *exec.Cmd
	restart := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Initial start
	restart <- true

	// Watcher goroutine
	go func() {
		lastSeen := make(map[string]time.Time)
		_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && filepath.Ext(path) == ".go" {
				lastSeen[path] = info.ModTime()
			}
			return nil
		})

		for {
			changed := false
			_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					if info.Name() == ".git" || info.Name() == "vendor" || info.Name() == "node_modules" {
						return filepath.SkipDir
					}
					return nil
				}
				if filepath.Ext(path) != ".go" {
					return nil
				}

				mtime := info.ModTime()
				if last, ok := lastSeen[path]; !ok || last.Before(mtime) {
					lastSeen[path] = mtime
					if ok {
						changed = true
					}
				}
				return nil
			})

			if changed {
				fmt.Printf("\n%s   📂  Change detected: restarting Flux...%s\n", colorBrightYellow, colorReset)
				restart <- true
			}
			time.Sleep(r.Interval)
		}
	}()

	killChild := func() {
		if child != nil && child.Process != nil {
			// Kill the entire process group
			pgid, err := syscall.Getpgid(child.Process.Pid)
			if err == nil {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
			} else {
				_ = child.Process.Signal(syscall.SIGTERM)
			}

			// Wait for exit with timeout
			done := make(chan struct{})
			go func() {
				_ = child.Wait()
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				// Force kill if still running
				if err == nil {
					_ = syscall.Kill(-pgid, syscall.SIGKILL)
				} else {
					_ = child.Process.Kill()
				}
			}
		}
	}

	initial := true
	for {
		select {
		case <-quit:
			killChild()
			return nil
		case <-restart:
			killChild()

			child = exec.Command(r.Args[0], r.Args[1:]...)
			child.Env = append(os.Environ(), "FLUX_HOT_RELOAD=true")
			if !initial {
				child.Env = append(child.Env, "FLUX_RESTARTED=true")
			}
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
			child.Stdin = os.Stdin
			// Set pgid so we can kill the whole tree later (Unix only)
			child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

			if err := child.Start(); err != nil {
				if initial {
					return fmt.Errorf("failed to start child process: %w", err)
				}
				fmt.Printf("%s   ❌  Restart failed: %v%s\n", colorRed, err, colorReset)
				continue
			}

			if initial {
				done := make(chan error, 1)
				go func() { done <- child.Wait() }()

				select {
				case err := <-done:
					if err != nil {
						return fmt.Errorf("initial process failed: %w", err)
					}
				case <-time.After(500 * time.Millisecond):
				}
			}
			initial = false
		}
	}
}
