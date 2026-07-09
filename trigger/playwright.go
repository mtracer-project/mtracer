package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	idgenerator "github.com/mtrace-project/mtrace/idGenerator"
	"github.com/mtrace-project/mtrace/parser"
)

const (
	PLAYWRIGHT_SERVER_ENDPOINT = "/traceparent"
	DEFAULT_PLAYWRIGHT_PATH    = "playwright"
	DEFAULT_URL_PATTERN        = "**"
)

type traceIdServer interface {
	start(traceIdChan chan<- TraceId) (*http.Server, error)
}

type PlaywrightTrigger struct {
	playwrightPath  string
	filePath        string
	projects        []string
	traceUrlPattern string

	server      traceIdServer
	baseDir     string
	ctx         context.Context
	idGenerator idgenerator.IdGenerator
}

func (t *PlaywrightTrigger) Init(dto *parser.TriggerDTO, idGenerator idgenerator.IdGenerator, baseDir string, ctx context.Context) error {
	filePath, ok := dto.Args["filePath"].(string)
	filePath = strings.TrimSpace(filePath)
	if !ok || filePath == "" {
		return fmt.Errorf("filePath is required")
	}

	playwrightPath, ok := dto.Args["playwrightPath"].(string)
	if !ok || playwrightPath == "" {
		playwrightPath = DEFAULT_PLAYWRIGHT_PATH
	}

	var projectList []string
	if projectsArg, ok := dto.Args["projects"]; ok {
		if vals, ok := projectsArg.([]any); ok {
			for _, project := range vals {
				if projectStr, ok := project.(string); ok {
					projectStr := strings.TrimSpace(projectStr)
					if projectStr != "" {
						projectList = append(projectList, projectStr)
					}
				}
			}
		}
	}

	traceUrlPattern, ok := dto.Args["traceUrlPattern"].(string)
	if !ok {
		traceUrlPattern = DEFAULT_URL_PATTERN
	}
	traceUrlPattern = strings.TrimSpace(traceUrlPattern)

	server := &playwrightTraceIdServer{
		serverAddress:   "localhost",
		port:            0, // the OS will assign an available port
		idGenerator:     idGenerator,
		ctx:             ctx,
		traceUrlPattern: traceUrlPattern,
	}

	t.playwrightPath = playwrightPath
	t.filePath = filePath
	t.projects = projectList
	t.traceUrlPattern = traceUrlPattern
	t.baseDir = baseDir
	t.server = server
	t.ctx = ctx
	t.idGenerator = idGenerator

	return nil
}

func (t *PlaywrightTrigger) Trigger() (TraceId, error) {
	var traceId TraceId
	traceIdChan := make(chan TraceId, 1)
	srv, err := t.server.start(traceIdChan)
	if err != nil {
		return "", fmt.Errorf("failed to start traceId server: %w", err)
	}
	defer srv.Shutdown(t.ctx) // nolint:errcheck

	args := []string{"playwright", "test", t.filePath}
	for _, proj := range t.projects {
		args = append(args, "--project", proj)
	}

	cmd := exec.CommandContext(t.ctx, "npx", args...)
	cmd.Dir = filepath.Join(t.baseDir, t.playwrightPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("MTRACE_PLAYWRIGHT_SERVER_URL=%s", fmt.Sprintf("http://%s%s", srv.Addr, PLAYWRIGHT_SERVER_ENDPOINT)))

	if slog.Default().Enabled(t.ctx, slog.LevelInfo) {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute Playwright test: %w", err)
	}

	select {
	case traceId = <-traceIdChan:
	default:
	}

	if traceId == "" {
		return "", fmt.Errorf("traceId was not set by the Playwright test")
	}

	slog.Info("Playwright test executed successfully", "filePath", t.filePath, "traceId", traceId)

	return traceId, nil
}

type playwrightTraceIdServer struct {
	serverAddress   string
	port            int
	idGenerator     idgenerator.IdGenerator
	ctx             context.Context
	traceUrlPattern string
}

type traceResponse struct {
	Traceparent     string `json:"traceparent"`
	TraceUrlPattern string `json:"traceUrlPattern"`
}

func (s *playwrightTraceIdServer) start(traceIdChan chan<- TraceId) (*http.Server, error) {
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(s.ctx, "tcp", fmt.Sprintf("%s:%d", s.serverAddress, s.port))
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(PLAYWRIGHT_SERVER_ENDPOINT, func(w http.ResponseWriter, r *http.Request) {
		tId, err := s.idGenerator.Generate(idgenerator.TRACE_ID_LENGTH)
		if err != nil {
			http.Error(w, "failed to generate trace ID", http.StatusInternalServerError)
			return
		}
		sId, err := s.idGenerator.Generate(idgenerator.SPAN_ID_LENGTH)
		if err != nil {
			http.Error(w, "failed to generate span ID", http.StatusInternalServerError)
			return
		}

		traceIdObj, err := NewTraceId(tId)
		if err != nil {
			http.Error(w, "failed to create trace ID", http.StatusInternalServerError)
			return
		}
		select {
		case traceIdChan <- traceIdObj:
		default:
		}

		tp := getTraceparent(tId, sId)
		resObj := traceResponse{
			Traceparent:     tp,
			TraceUrlPattern: s.traceUrlPattern,
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resObj)
		if err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
			return
		}
	})

	srv := &http.Server{Handler: mux}
	srv.Addr = listener.Addr().String()

	go func() {
		_ = srv.Serve(listener) // nolint:errcheck
	}()
	return srv, nil
}

func (t *PlaywrightTrigger) Example() string {
	return `trigger:
  type: "playwright"
  args:
    filePath: "example.spec.ts"`
}
