package integrationtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type readyzResp struct {
	Ok     bool              `json:"ok"`
	Checks map[string]string `json:"checks"`
}

type httpStatusError struct {
	code int
	body string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("http status %d: %s", e.code, e.body)
}

func waitHTTP(addr string, path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	url := "http://" + addr + path
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == 200 {
			return b, nil
		}
		lastErr = &httpStatusError{code: resp.StatusCode, body: string(b)}
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	return nil, lastErr
}

func freeAddr(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestServer_Readyz_MemoryRedis_EmbeddedNATS(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	if os.Getenv("PROJECT_TEMPLATE_INTEGRATION") != "1" {
		t.Skip("set PROJECT_TEMPLATE_INTEGRATION=1 to run integration tests")
	}
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows in CI")
	}

	// This integration test expects local postgres to be up at 127.0.0.1:5432.
	// Run: make infra-up && make migrate-dev
	if c, err := net.DialTimeout("tcp", "127.0.0.1:5432", 400*time.Millisecond); err != nil {
		t.Fatalf("postgres is not reachable at 127.0.0.1:5432 (run `make infra-up`): %v", err)
	} else {
		_ = c.Close()
	}

	addr := freeAddr(t)
	grpcAddr := freeAddr(t)

	root := filepath.Clean("../..")
	defaultConfPath := filepath.Join(root, "docker", "backend", "config", "config.yaml")
	configBytes, err := os.ReadFile(defaultConfPath)
	if err != nil {
		t.Fatalf("read default config: %v", err)
	}

	content := string(configBytes)
	content = strings.ReplaceAll(content, "addr: 0.0.0.0:8080", "addr: "+addr)
	content = strings.ReplaceAll(content, "addr: 0.0.0.0:8081", "addr: "+grpcAddr)
	content = strings.ReplaceAll(content, "mode: \"external\"", "mode: \"memory\"")
	content = strings.ReplaceAll(content, "addr: \"redis:6379\"", "addr: \"\"")
	content = strings.ReplaceAll(content, "host=postgres", "host=127.0.0.1")

	// Add NATS config if missing
	if !strings.Contains(content, "nats:") {
		content += "\nnats:\n  mode: \"embedded\"\n  url: \"\"\n  jetstream:\n    enabled: true\n"
	}

	tempDir := t.TempDir()
	tempConf := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(tempConf, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/server/main.go", "--conf", tempConf)
	cmd.Dir = root

	logPath := filepath.Join(t.TempDir(), "projecttemplate.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("create log: %v", err)
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	body, err := waitHTTP(addr, "/readyz", 20*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		b, _ := os.ReadFile(logPath)
		t.Fatalf("readyz: %v\n--- server log ---\n%s", err, string(b))
	}

	var rr readyzResp
	if err := json.Unmarshal(body, &rr); err != nil {
		t.Fatalf("unmarshal readyz: %v body=%s", err, string(body))
	}
	if !rr.Ok {
		t.Fatalf("expected ok=true, got %v (%v)", rr.Ok, rr.Checks)
	}
	if rr.Checks["nats"] != "ok" {
		t.Fatalf("expected nats ok, got %q", rr.Checks["nats"])
	}
	if rr.Checks["jetstream"] != "ok" {
		t.Fatalf("expected jetstream ok, got %q", rr.Checks["jetstream"])
	}
}
