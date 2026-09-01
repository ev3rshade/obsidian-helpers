package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Runner struct {
	dir string
	env []string
	timeout time.Duration
	stream bool
}

type Result struct {
	Stdout string
	Stderr string
	ExitCode int
	Duration time.Duration
}

type Option func(*Runner) // a named function type. for readability

func WithDir(d string) Option {
	return func(r *Runner) { r.dir = d }
}

func WithEnv(kv ...string) Option {
	return func(r *Runner) { r.env = append(r.env, kv ...) }
}

func WithTimeout(d time.Duration) Option {
	return func(r *Runner) { r.timeout = d }
}

func WithStreaming() Option {
	return func(r *Runner) { r.stream = true }
}

func New(opts ...Option) * Runner {
	r:= &Runner {
		timeout: 30 * time.Second,
		env: os.Environ(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Runner) Run(ctx context.Context, name string, args ...string) (*Result, error) {

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel() // to cancel via context if the command runs too long

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = r.dir
	cmd.Env = r.env

	var stdout, stderr bytes.Buffer
	if r.stream {
		cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
		cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	start := time.Now()
	err := cmd.Run()

	res := &Result {
		Stdout: strings.TrimRight(stdout.String(), "\n"),
		Stderr: strings.TrimRight(stderr.String(), "\n"),
		Duration: time.Since(start),
	}

	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, fmt.Errorf("%s exited %d: %s", name, res.ExitCode, res.Stderr)
		}
		return res, fmt.Errorf("run %s: %w", name, err)
	}
	return res, nil
}