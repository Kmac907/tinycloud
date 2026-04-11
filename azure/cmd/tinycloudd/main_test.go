package main

import (
	"bytes"
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

func TestRepoRootTinyClouddScriptServesHealthEndpoint(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("tinycloudd script test requires Windows")
	}

	powerShellExe, err := exec.LookPath("pwsh")
	if err != nil {
		powerShellExe, err = exec.LookPath("powershell")
		if err != nil {
			t.Fatalf("resolve PowerShell: %v", err)
		}
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)
	scriptPath := filepath.Join(repoRoot, "scripts", "tinycloudd.ps1")

	httpPort := reserveTCPPort(t)
	httpsPort := reserveTCPPort(t)
	blobPort := reserveTCPPort(t)
	queuePort := reserveTCPPort(t)
	tablePort := reserveTCPPort(t)
	keyVaultPort := reserveTCPPort(t)
	serviceBusPort := reserveTCPPort(t)
	appConfigPort := reserveTCPPort(t)
	cosmosPort := reserveTCPPort(t)
	eventHubsPort := reserveTCPPort(t)
	dnsPort := reserveUDPPort(t)

	dataRoot := t.TempDir()
	runtimeRoot := filepath.Join(azureRoot, ".verify-root-tinycloudd-runtime")
	_ = os.RemoveAll(runtimeRoot)

	cmd := exec.Command(powerShellExe, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.Env = append(os.Environ(),
		"GOCACHE="+filepath.Join(azureRoot, ".gocache"),
		"TINYCLOUD_RUNTIME_ROOT="+runtimeRoot,
		"TINYCLOUD_DATA_ROOT="+dataRoot,
		"TINYCLOUD_LISTEN_HOST=127.0.0.1",
		"TINYCLOUD_ADVERTISE_HOST=127.0.0.1",
		"TINYCLOUD_MGMT_HTTP_PORT="+httpPort,
		"TINYCLOUD_MGMT_HTTPS_PORT="+httpsPort,
		"TINYCLOUD_BLOB_PORT="+blobPort,
		"TINYCLOUD_QUEUE_PORT="+queuePort,
		"TINYCLOUD_TABLE_PORT="+tablePort,
		"TINYCLOUD_KEYVAULT_PORT="+keyVaultPort,
		"TINYCLOUD_SERVICEBUS_PORT="+serviceBusPort,
		"TINYCLOUD_APPCONFIG_PORT="+appConfigPort,
		"TINYCLOUD_COSMOS_PORT="+cosmosPort,
		"TINYCLOUD_DNS_PORT="+dnsPort,
		"TINYCLOUD_EVENTHUBS_PORT="+eventHubsPort,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start() error = %v", err)
	}
	t.Cleanup(func() {
		stopProcessListeningOnTCPPort(t, httpPort)
		killProcessIfRunning(t, cmd.Process)
		waitForCommandExit(t, cmd)
		removePathWithRetries(t, runtimeRoot)
	})

	healthURL := fmt.Sprintf("http://127.0.0.1:%s/_admin/healthz", httpPort)
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr == nil && resp.StatusCode == http.StatusOK && strings.Contains(string(body), "ok") {
				if _, statErr := os.Stat(filepath.Join(runtimeRoot, "tinycloudd.exe")); statErr != nil {
					t.Fatalf("tinycloudd.exe was not built in runtime root: %v", statErr)
				}
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
		if strings.Contains(strings.ToLower(stderr.String()), "access is denied") {
			t.Skipf("repo-root tinycloudd script runtime is blocked in this environment: %s", stderr.String())
		}
	}

	if strings.Contains(strings.ToLower(stderr.String()), "access is denied") {
		t.Skipf("repo-root tinycloudd script runtime is blocked in this environment: %s", stderr.String())
	}
	t.Fatalf("health check never succeeded: url=%s stdout=%q stderr=%q", healthURL, stdout.String(), stderr.String())
}

func TestRepoRootGoRunTopLevelTinyClouddServesHealthEndpoint(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("repo-root go run test requires Windows")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	azureRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoRoot := filepath.Dir(azureRoot)

	httpPort := reserveTCPPort(t)
	httpsPort := reserveTCPPort(t)
	blobPort := reserveTCPPort(t)
	queuePort := reserveTCPPort(t)
	tablePort := reserveTCPPort(t)
	keyVaultPort := reserveTCPPort(t)
	serviceBusPort := reserveTCPPort(t)
	appConfigPort := reserveTCPPort(t)
	cosmosPort := reserveTCPPort(t)
	eventHubsPort := reserveTCPPort(t)
	dnsPort := reserveUDPPort(t)

	dataRoot := t.TempDir()
	cmd := exec.Command("go", "run", ".\\cmd\\tinycloudd")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"GOCACHE="+filepath.Join(t.TempDir(), "gocache"),
		"TINYCLOUD_DATA_ROOT="+dataRoot,
		"TINYCLOUD_LISTEN_HOST=127.0.0.1",
		"TINYCLOUD_ADVERTISE_HOST=127.0.0.1",
		"TINYCLOUD_MGMT_HTTP_PORT="+httpPort,
		"TINYCLOUD_MGMT_HTTPS_PORT="+httpsPort,
		"TINYCLOUD_BLOB_PORT="+blobPort,
		"TINYCLOUD_QUEUE_PORT="+queuePort,
		"TINYCLOUD_TABLE_PORT="+tablePort,
		"TINYCLOUD_KEYVAULT_PORT="+keyVaultPort,
		"TINYCLOUD_SERVICEBUS_PORT="+serviceBusPort,
		"TINYCLOUD_APPCONFIG_PORT="+appConfigPort,
		"TINYCLOUD_COSMOS_PORT="+cosmosPort,
		"TINYCLOUD_DNS_PORT="+dnsPort,
		"TINYCLOUD_EVENTHUBS_PORT="+eventHubsPort,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start() error = %v", err)
	}
	t.Cleanup(func() {
		stopProcessListeningOnTCPPort(t, httpPort)
		killProcessIfRunning(t, cmd.Process)
		waitForCommandExit(t, cmd)
	})

	healthURL := fmt.Sprintf("http://127.0.0.1:%s/_admin/healthz", httpPort)
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr == nil && resp.StatusCode == http.StatusOK && strings.Contains(string(body), "ok") {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
		if strings.Contains(strings.ToLower(stderr.String()), "access is denied") {
			t.Skipf("repo-root top-level tinycloudd go run is blocked in this environment: %s", stderr.String())
		}
	}

	if strings.Contains(strings.ToLower(stderr.String()), "access is denied") {
		t.Skipf("repo-root top-level tinycloudd go run is blocked in this environment: %s", stderr.String())
	}
	t.Fatalf("health check never succeeded: url=%s stdout=%q stderr=%q", healthURL, stdout.String(), stderr.String())
}

func reserveTCPPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve TCP port: %v", err)
	}
	defer listener.Close()
	return fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
}

func reserveUDPPort(t *testing.T) string {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve UDP port: %v", err)
	}
	defer conn.Close()
	return fmt.Sprintf("%d", conn.LocalAddr().(*net.UDPAddr).Port)
}

func stopProcessListeningOnTCPPort(t *testing.T, port string) {
	t.Helper()
	command := fmt.Sprintf(`Get-NetTCPConnection -LocalPort %s -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess -Unique | ForEach-Object { Stop-Process -Id $_ -Force -ErrorAction SilentlyContinue }`, port)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", command)
	if output, err := cmd.CombinedOutput(); err != nil && strings.TrimSpace(string(output)) != "" {
		t.Fatalf("stop process on TCP port %s failed: %v output=%q", port, err, string(output))
	}
}

func killProcessIfRunning(t *testing.T, process *os.Process) {
	t.Helper()
	if process == nil {
		return
	}
	if err := process.Kill(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "finished") {
		// The server process is the important one; killing the wrapper shell is best-effort.
	}
}

func waitForCommandExit(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd == nil || cmd.Process == nil {
		return
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(10 * time.Second):
	case <-done:
	}
}

func removePathWithRetries(t *testing.T, path string) {
	t.Helper()
	for i := 0; i < 20; i++ {
		err := os.RemoveAll(path)
		if err == nil || os.IsNotExist(err) {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}
