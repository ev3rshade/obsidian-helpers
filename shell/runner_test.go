package shell

import (
	"context"
	"testing"
	"time"
)

func TestOptions(t *testing.T) {
	r := New(
		WithDir("/tmp"),
		WithTimeout(5*time.Second),
		WithStreaming(),
	)

	if r.dir != "/tmp" {
		t.Errorf("dir = %q, want %q", r.dir, "/tmp")
	}
	if r.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want %v", r.timeout, 5*time.Second)
	}
	if !r.stream {
		t.Error("stream = false, want true")
	}
}

func TestRunTimeout(t *testing.T) {
    r := New(WithTimeout(100 * time.Millisecond))

    start := time.Now()
    _, err := r.Run(context.Background(), "sleep", "5")
    elapsed := time.Since(start)

    if err == nil {
        t.Fatal("Run() error = nil, want timeout error")
    }
    if elapsed > time.Second {
        t.Errorf("took %v, want the timeout to have killed it", elapsed)
    }
}

func TestRunContextCancel(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())

    go func() {
        time.Sleep(50 * time.Millisecond)
        cancel()
    }()

    r := New()
    if _, err := r.Run(ctx, "sleep", "5"); err == nil {
        t.Fatal("Run() error = nil, want cancellation error")
    }
}

// starter example test
func TestRunCapturesStdout(t *testing.T) {
	r := New()
	res, err := r.Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("Run() return error: %v", err)
	}
	if res.Stdout != "hello" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "hello")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		args     []string
		wantOut  string
		wantCode int
		wantErr  bool
	}{
		{
			name:    "simple echo",
			cmd:     "echo",
			args:    []string{"hello"},
			wantOut: "hello",
		},
		{
			name:    "multiple args",
			cmd:     "echo",
			args:    []string{"a", "b", "c"},
			wantOut: "a b c",
		},
		{
			name:     "nonzero exit",
			cmd:      "false",
			wantCode: 1,
			wantErr:  true,
		},
		{
			name:    "binary not found",
			cmd:     "definitely-not-a-real-binary",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			res, err := r.Run(context.Background(), tt.cmd, tt.args...)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if res.Stdout != tt.wantOut {
				t.Errorf("Stdout = %q, want %q", res.Stdout, tt.wantOut)
			}
			if res.ExitCode != tt.wantCode {
				t.Errorf("ExitCode = %d, want %d", res.ExitCode, tt.wantCode)
			}
		})
	}
}
