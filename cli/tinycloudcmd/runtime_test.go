package tinycloudcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tinycloud/runtime/tinycloudconfig"
)

func TestLoadRuntimeRecordRebuildsConfigFromEnv(t *testing.T) {
	t.Parallel()

	runtimeRoot := t.TempDir()
	record := runtimeRecord{
		Backend: "process",
		Env: map[string]string{
			"TINYCLOUD_SERVICES":        "management,storage",
			"TINYCLOUD_LISTEN_HOST":     "127.0.0.1",
			"TINYCLOUD_ADVERTISE_HOST":  "127.0.0.1",
			"TINYCLOUD_MGMT_HTTP_PORT":  "4566",
			"TINYCLOUD_MGMT_HTTPS_PORT": "4567",
			"TINYCLOUD_BLOB_PORT":       "4577",
			"TINYCLOUD_QUEUE_PORT":      "4578",
			"TINYCLOUD_TABLE_PORT":      "4579",
		},
	}
	if err := saveRuntimeRecord(runtimeRoot, record); err != nil {
		t.Fatalf("saveRuntimeRecord() error = %v", err)
	}

	loaded, err := loadRuntimeRecord(runtimeRoot)
	if err != nil {
		t.Fatalf("loadRuntimeRecord() error = %v", err)
	}

	if !loaded.Config.ServiceEnabled(tinycloudconfig.ServiceManagement) ||
		!loaded.Config.ServiceEnabled(tinycloudconfig.ServiceBlob) ||
		!loaded.Config.ServiceEnabled(tinycloudconfig.ServiceQueue) ||
		!loaded.Config.ServiceEnabled(tinycloudconfig.ServiceTable) {
		t.Fatalf("loaded config lost enabled services: %#v", loaded.Config.EnabledServices())
	}
}

func TestRunEServicesDisablePrintsRestartGuidanceWhenRuntimeActive(t *testing.T) {
	t.Parallel()

	repoRoot := repoRootFromTestFile(t)
	runtimeRoot := filepath.Join(t.TempDir(), "tinycloud-runtime")
	if err := saveStoredEnv(runtimeRoot, map[string]string{
		"TINYCLOUD_RUNTIME_ROOT": runtimeRoot,
	}); err != nil {
		t.Fatalf("saveStoredEnv() error = %v", err)
	}
	if err := saveRuntimeRecord(runtimeRoot, runtimeRecord{
		Backend:     "process",
		PID:         os.Getpid(),
		StartedAt:   "2026-04-10T00:00:00Z",
		RepoRoot:    repoRoot,
		RuntimeRoot: runtimeRoot,
		Env: map[string]string{
			"TINYCLOUD_RUNTIME_ROOT": runtimeRoot,
			"TINYCLOUD_SERVICES":     "management,storage,messaging",
		},
	}); err != nil {
		t.Fatalf("saveRuntimeRecord() error = %v", err)
	}

	oldRuntimeRoot := os.Getenv("TINYCLOUD_RUNTIME_ROOT")
	t.Cleanup(func() {
		if oldRuntimeRoot == "" {
			_ = os.Unsetenv("TINYCLOUD_RUNTIME_ROOT")
			return
		}
		_ = os.Setenv("TINYCLOUD_RUNTIME_ROOT", oldRuntimeRoot)
	})
	if err := os.Setenv("TINYCLOUD_RUNTIME_ROOT", runtimeRoot); err != nil {
		t.Fatalf("Setenv(TINYCLOUD_RUNTIME_ROOT) error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code, err := RunE([]string{"services", "disable", "storage"}, &stdout, &stderr, func() (string, error) {
		return repoRoot, nil
	})
	if err != nil {
		t.Fatalf("RunE() error = %v, stderr = %q", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("RunE() code = %d, want 0, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	for _, fragment := range []string{
		"Service Selection Updated",
		"services  ● management,serviceBus,eventHubs",
		"restart   ‼ required",
		"tinycloud restart",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("output missing %q in %q", fragment, output)
		}
	}
}

func TestRunEServicesListAndStatusServicesHaveDistinctJSONShapes(t *testing.T) {
	t.Parallel()

	repoRoot := repoRootFromTestFile(t)
	runtimeRoot := filepath.Join(t.TempDir(), "tinycloud-runtime")

	oldRuntimeRoot := os.Getenv("TINYCLOUD_RUNTIME_ROOT")
	oldServices := os.Getenv("TINYCLOUD_SERVICES")
	t.Cleanup(func() {
		if oldRuntimeRoot == "" {
			_ = os.Unsetenv("TINYCLOUD_RUNTIME_ROOT")
		} else {
			_ = os.Setenv("TINYCLOUD_RUNTIME_ROOT", oldRuntimeRoot)
		}
		if oldServices == "" {
			_ = os.Unsetenv("TINYCLOUD_SERVICES")
		} else {
			_ = os.Setenv("TINYCLOUD_SERVICES", oldServices)
		}
	})
	if err := os.Setenv("TINYCLOUD_RUNTIME_ROOT", runtimeRoot); err != nil {
		t.Fatalf("Setenv(TINYCLOUD_RUNTIME_ROOT) error = %v", err)
	}
	if err := os.Setenv("TINYCLOUD_SERVICES", "management,storage"); err != nil {
		t.Fatalf("Setenv(TINYCLOUD_SERVICES) error = %v", err)
	}

	var listStdout bytes.Buffer
	var listStderr bytes.Buffer
	code, err := RunE([]string{"services", "list", "--json"}, &listStdout, &listStderr, func() (string, error) {
		return repoRoot, nil
	})
	if err != nil {
		t.Fatalf("RunE(services list) error = %v, stderr = %q", err, listStderr.String())
	}
	if code != 0 {
		t.Fatalf("RunE(services list) code = %d, stderr = %q", code, listStderr.String())
	}

	var listBody struct {
		Services []map[string]any `json:"services"`
	}
	if err := json.Unmarshal(listStdout.Bytes(), &listBody); err != nil {
		t.Fatalf("json.Unmarshal(services list) error = %v", err)
	}
	if len(listBody.Services) == 0 || listBody.Services[0]["family"] == nil {
		t.Fatalf("services list json missing family field: %s", listStdout.String())
	}
	if _, ok := listBody.Services[0]["status"]; ok {
		t.Fatalf("services list json unexpectedly contained status field: %s", listStdout.String())
	}

	var statusStdout bytes.Buffer
	var statusStderr bytes.Buffer
	code, err = RunE([]string{"status", "services", "--json"}, &statusStdout, &statusStderr, func() (string, error) {
		return repoRoot, nil
	})
	if err != nil {
		t.Fatalf("RunE(status services) error = %v, stderr = %q", err, statusStderr.String())
	}
	if code != 0 {
		t.Fatalf("RunE(status services) code = %d, stderr = %q", code, statusStderr.String())
	}

	var statusBody struct {
		Services []map[string]any `json:"services"`
	}
	if err := json.Unmarshal(statusStdout.Bytes(), &statusBody); err != nil {
		t.Fatalf("json.Unmarshal(status services) error = %v", err)
	}
	if len(statusBody.Services) == 0 || statusBody.Services[0]["status"] == nil {
		t.Fatalf("status services json missing status field: %s", statusStdout.String())
	}
	if _, ok := statusBody.Services[0]["family"]; ok {
		t.Fatalf("status services json unexpectedly contained family field: %s", statusStdout.String())
	}
}

func repoRootFromTestFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
