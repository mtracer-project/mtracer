package dockersetupcommand_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	dockersetupcommand "github.com/mtrace-project/mtrace/setupCommand/docker"
	testutils "github.com/mtrace-project/mtrace/testUtils"

	"github.com/moby/moby/client"
)

func TestNewDockerSetupCommand(t *testing.T) {
	t.Run("killcontainer", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "killcontainer",
			Args: map[string]any{"containerId": "test"},
		}
		cmd, err := dockersetupcommand.NewDockerSetupCommand(dto, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("stopcontainer", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "stopcontainer",
			Args: map[string]any{"containerId": "test"},
		}
		cmd, err := dockersetupcommand.NewDockerSetupCommand(dto, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("startcontainer", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "startcontainer",
			Args: map[string]any{"containerId": "test"},
		}
		cmd, err := dockersetupcommand.NewDockerSetupCommand(dto, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("pausecontainer", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "pausecontainer",
			Args: map[string]any{"containerId": "test"},
		}
		cmd, err := dockersetupcommand.NewDockerSetupCommand(dto, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("unpausecontainer", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "unpausecontainer",
			Args: map[string]any{"containerId": "test"},
		}
		cmd, err := dockersetupcommand.NewDockerSetupCommand(dto, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("execcontainer", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "execcontainer",
			Args: map[string]any{"containerId": "test-id", "cmd": "ls"},
			CleanupCmd: &parser.CleanupCommandDTO{
				Cmd: "echo 'clean'",
			},
		}
		cmd, err := dockersetupcommand.NewDockerSetupCommand(dto, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("composeup", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "composeup",
			Args: map[string]any{"composePath": "docker-compose.yml"},
		}
		handler := dockersetupcommand.NewDockerHandler(nil, "/base", context.Background())
		cmd, err := dockersetupcommand.NewDockerSetupCommand(dto, handler)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("unsupported docker command type", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Type: "docker",
			Cmd:  "unsupported_action",
		}
		_, err := dockersetupcommand.NewDockerSetupCommand(dto, nil)
		if err == nil || !strings.Contains(err.Error(), "unsupported docker setup command type") {
			t.Errorf("expected error about unsupported type, got %v", err)
		}
	})
}

func TestDockerHandler_Execute_DebugDisabled(t *testing.T) {
	var calledCreate, calledStart bool

	cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container/exec") {
			calledCreate = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"Id":"exec-123"}`))),
			}, nil
		}
		if req.Method == "POST" && strings.Contains(req.URL.Path, "/exec/exec-123/start") {
			calledStart = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
		}, nil
	})

	// Disable info logging for this test to ensure attach is not called
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(oldLogger)

	executor := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
	err := executor.Execute("test-container", "ls")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !calledCreate || !calledStart {
		t.Errorf("expected exec create and start to be called. create: %t, start: %t", calledCreate, calledStart)
	}
}

func TestDockerHandler_Execute_DebugEnabled(t *testing.T) {
	var calledCreate, calledAttach, calledStart bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/containers/test-container/exec") {
			calledCreate = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"Id":"exec-123"}`)) // nolint: errcheck
			return
		}

		if r.Method == "POST" && strings.Contains(r.URL.Path, "/exec/exec-123/start") {
			// Check if this is an attach/upgrade request
			if strings.ToLower(r.Header.Get("Connection")) == "upgrade" || strings.ToLower(r.Header.Get("Upgrade")) == "tcp" {
				calledAttach = true
				hj, ok := w.(http.Hijacker)
				if !ok {
					http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
					return
				}
				conn, bufrw, err := hj.Hijack()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: tcp\r\nConnection: Upgrade\r\n\r\n") // nolint: errcheck
				// 8-byte multiplex header: stdout type 1, length 11 ("hello world")
				header := []byte{1, 0, 0, 0, 0, 0, 0, 11}
				payload := []byte("hello world")
				bufrw.Write(header)  // nolint: errcheck
				bufrw.Write(payload) // nolint: errcheck
				bufrw.Flush()        // nolint: errcheck

				if cw, ok := conn.(interface{ CloseWrite() error }); ok {
					cw.CloseWrite() // nolint: errcheck
				}
				io.Copy(io.Discard, conn) // nolint: errcheck
				conn.Close()              // nolint: errcheck
				return
			} else {
				calledStart = true
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{}")) // nolint: errcheck
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cli, err := client.New(
		client.WithHost("tcp://"+server.Listener.Addr().String()),
		client.WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Enable debug logging for this test
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(oldLogger)

	executor := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
	err = executor.Execute("test-container", "ls")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !calledCreate || !calledAttach || !calledStart {
		t.Errorf("expected exec create, attach, and start to be called. create: %t, attach: %t, start: %t", calledCreate, calledAttach, calledStart)
	}
}

func TestDockerHandler_Execute_ErrorCases(t *testing.T) {
	t.Run("ExecCreate error", func(t *testing.T) {
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("internal server error")),
			}, nil
		})

		executor := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := executor.Execute("test-container", "ls")
		if err == nil || !strings.Contains(err.Error(), "failed to create exec in container") {
			t.Errorf("expected error about failed exec create, got %v", err)
		}
	})

	t.Run("ExecStart error", func(t *testing.T) {
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container/exec") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"Id":"exec-123"}`))),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("internal server error")),
			}, nil
		})

		// Disable info logging for this test to ensure attach is not called
		oldLogger := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn})))
		defer slog.SetDefault(oldLogger)

		executor := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := executor.Execute("test-container", "ls")
		if err == nil || !strings.Contains(err.Error(), "failed to start exec in container") {
			t.Errorf("expected error about failed exec start, got %v", err)
		}
	})

	t.Run("ExecAttach error (debug enabled)", func(t *testing.T) {
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container/exec") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"Id":"exec-123"}`))),
				}, nil
			}
			// Let start/attach fail
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("internal server error")),
			}, nil
		})

		oldLogger := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})))
		defer slog.SetDefault(oldLogger)

		executor := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := executor.Execute("test-container", "ls")
		if err == nil || !strings.Contains(err.Error(), "failed to attach to exec in container") {
			t.Errorf("expected error about failed exec attach, got %v", err)
		}
	})
}

func TestDockerHandler_Start(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var called bool
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container/start") {
				called = true
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("{}")),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Start("test-container")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected ContainerStart to be called")
		}
	})

	t.Run("error", func(t *testing.T) {
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("internal server error")),
			}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Start("test-container")
		if err == nil || !strings.Contains(err.Error(), "failed to start container") {
			t.Errorf("expected error about failed start, got %v", err)
		}
	})
}

func TestDockerHandler_Stop(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var called bool
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container/stop") {
				called = true
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("{}")),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Stop("test-container")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected ContainerStop to be called")
		}
	})

	t.Run("error", func(t *testing.T) {
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("internal server error")),
			}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Stop("test-container")
		if err == nil || !strings.Contains(err.Error(), "failed to stop container") {
			t.Errorf("expected error about failed stop, got %v", err)
		}
	})
}

func TestDockerHandler_Kill(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var called bool
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container/kill") {
				called = true
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("{}")),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Kill("test-container")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected ContainerKill to be called")
		}
	})

	t.Run("error", func(t *testing.T) {
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("internal server error")),
			}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Kill("test-container")
		if err == nil || !strings.Contains(err.Error(), "failed to kill container") {
			t.Errorf("expected error about failed kill, got %v", err)
		}
	})
}

func TestDockerHandler_Pause(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var called bool
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container/pause") {
				called = true
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("{}")),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Pause("test-container")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected ContainerPause to be called")
		}
	})

	t.Run("error", func(t *testing.T) {
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("internal server error")),
			}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Pause("test-container")
		if err == nil || !strings.Contains(err.Error(), "failed to pause container") {
			t.Errorf("expected error about failed pause, got %v", err)
		}
	})
}

func TestDockerHandler_Unpause(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var called bool
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container/unpause") {
				called = true
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("{}")),
				}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Unpause("test-container")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected ContainerUnpause to be called")
		}
	})

	t.Run("error", func(t *testing.T) {
		cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("internal server error")),
			}, nil
		})

		handler := dockersetupcommand.NewDockerHandler(cli, "", context.Background())
		err := handler.Unpause("test-container")
		if err == nil || !strings.Contains(err.Error(), "failed to unpause container") {
			t.Errorf("expected error about failed unpause, got %v", err)
		}
	})
}
