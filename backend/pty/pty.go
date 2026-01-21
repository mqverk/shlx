package pty

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// Terminal manages a pseudo-terminal
type Terminal struct {
	ptmx   *os.File
	cmd    *exec.Cmd
	mu     sync.Mutex
	closed bool
}

// New creates a new PTY with the given shell
func New(shell string, args []string) (*Terminal, error) {
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
	}

	cmd := exec.Command(shell, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return &Terminal{
		ptmx: ptmx,
		cmd:  cmd,
	}, nil
}

// Write writes data to the PTY
func (t *Terminal) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return 0, io.ErrClosedPipe
	}
	return t.ptmx.Write(p)
}

// Read reads data from the PTY
func (t *Terminal) Read(p []byte) (int, error) {
	return t.ptmx.Read(p)
}

// Resize changes the PTY size
func (t *Terminal) Resize(rows, cols uint16) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return io.ErrClosedPipe
	}

	size := &pty.Winsize{
		Rows: rows,
		Cols: cols,
	}
	return pty.Setsize(t.ptmx, size)
}

// Close terminates the PTY and the shell process
func (t *Terminal) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	// Send SIGTERM to process group
	if t.cmd.Process != nil {
		pgid, err := syscall.Getpgid(t.cmd.Process.Pid)
		if err == nil {
			syscall.Kill(-pgid, syscall.SIGTERM)
		}
	}

	t.ptmx.Close()
	t.cmd.Wait()
	return nil
}
