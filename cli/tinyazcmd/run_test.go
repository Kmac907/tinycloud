package tinyazcmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunERequiresArguments(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code, err := RunE(nil, strings.NewReader(""), &stdout, &stderr, os.Getwd, nil, time.Now, nil)
	if err == nil {
		t.Fatal("RunE() error = nil, want usage error")
	}
	if code != 2 {
		t.Fatalf("RunE() code = %d, want %d", code, 2)
	}
	if !strings.Contains(stdout.String(), "tinyaz commands:") {
		t.Fatalf("RunE() stdout = %q, want usage text", stdout.String())
	}
}

func TestRunEVersionPassesThroughToAz(t *testing.T) {
	t.Parallel()

	azExe := writeFakeAz(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code, err := RunE([]string{"version"}, strings.NewReader(""), &stdout, &stderr, os.Getwd, nil, time.Now, func(string) (string, error) {
		return azExe, nil
	})
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("RunE() code = %d, want %d", code, 0)
	}
	got := strings.TrimSpace(stdout.String())
	if got != `{"argv":"version"}` {
		t.Fatalf("stdout = %q, want passthrough output", got)
	}
}

func TestRunEAccountShowOutputsExpectedShape(t *testing.T) {
	t.Parallel()

	loader := func(cwd string, required []string) (map[string]string, error) {
		return map[string]string{
			"ARM_SUBSCRIPTION_ID": "sub-123",
			"ARM_TENANT_ID":       "tenant-456",
		}, nil
	}

	var stdout bytes.Buffer
	code, err := RunE([]string{"account", "show"}, strings.NewReader(""), &stdout, io.Discard, os.Getwd, loader, time.Now, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("RunE() code = %d, want %d", code, 0)
	}

	var payload accountOutput
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload.ID != "sub-123" || payload.TenantID != "tenant-456" {
		t.Fatalf("account payload = %+v", payload)
	}
	if payload.User.Name != "tinycloud" || payload.User.Type != "servicePrincipal" {
		t.Fatalf("account user = %+v", payload.User)
	}
}

func TestRunEAccountListOutputsArray(t *testing.T) {
	t.Parallel()

	loader := func(cwd string, required []string) (map[string]string, error) {
		return map[string]string{
			"ARM_SUBSCRIPTION_ID": "sub-123",
			"ARM_TENANT_ID":       "tenant-456",
		}, nil
	}

	var stdout bytes.Buffer
	code, err := RunE([]string{"account", "list"}, strings.NewReader(""), &stdout, io.Discard, os.Getwd, loader, time.Now, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("RunE() code = %d, want %d", code, 0)
	}

	var payload []accountOutput
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("len(payload) = %d, want 1", len(payload))
	}
	if payload[0].ID != "sub-123" {
		t.Fatalf("payload[0].ID = %q, want %q", payload[0].ID, "sub-123")
	}
}

func TestRunEAccountGetAccessTokenUsesScope(t *testing.T) {
	t.Parallel()

	var requestedResource string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		requestedResource = r.Form.Get("resource")
		_, _ = io.WriteString(w, `{"access_token":"token-abc"}`)
	}))
	defer server.Close()

	loader := func(cwd string, required []string) (map[string]string, error) {
		return map[string]string{
			"ARM_SUBSCRIPTION_ID": "sub-123",
			"ARM_TENANT_ID":       "tenant-456",
			"TINY_OAUTH_TOKEN":    server.URL,
		}, nil
	}

	fixedNow := func() time.Time {
		return time.Unix(1_700_000_000, 0).UTC()
	}

	var stdout bytes.Buffer
	code, err := RunE([]string{"account", "get-access-token", "--scope", "https://storage.azure.com/.default"}, strings.NewReader(""), &stdout, io.Discard, os.Getwd, loader, fixedNow, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("RunE() code = %d, want %d", code, 0)
	}
	if requestedResource != "https://storage.azure.com" {
		t.Fatalf("requested resource = %q, want %q", requestedResource, "https://storage.azure.com")
	}

	var payload tokenOutput
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload.AccessToken != "token-abc" {
		t.Fatalf("accessToken = %q, want %q", payload.AccessToken, "token-abc")
	}
	if payload.Subscription != "sub-123" || payload.Tenant != "tenant-456" {
		t.Fatalf("token payload = %+v", payload)
	}
	if payload.ExpiresUnix != fixedNow().Add(time.Hour).Unix() {
		t.Fatalf("expires_on = %d, want %d", payload.ExpiresUnix, fixedNow().Add(time.Hour).Unix())
	}
}

func TestTokenResourceDefaultsToManagement(t *testing.T) {
	t.Parallel()

	if got := tokenResource(nil); got != "https://management.azure.com/" {
		t.Fatalf("tokenResource(nil) = %q, want default management resource", got)
	}
}

func TestRunEPassesUnsupportedCommandsThroughToAz(t *testing.T) {
	t.Parallel()

	azExe := writeFakeAz(t)
	var stdout bytes.Buffer
	code, err := RunE([]string{"group", "list"}, strings.NewReader(""), &stdout, io.Discard, os.Getwd, nil, time.Now, func(string) (string, error) {
		return azExe, nil
	})
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("RunE() code = %d, want %d", code, 0)
	}
	if got := strings.TrimSpace(stdout.String()); got != `{"argv":"group list"}` {
		t.Fatalf("stdout = %q, want passthrough output", got)
	}
}

func TestResolveAzExeHonorsOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-az")
	t.Setenv("TINYAZ_EXE", override)

	got, err := ResolveAzExe(nil)
	if err != nil {
		t.Fatalf("ResolveAzExe() error = %v", err)
	}
	if got != override {
		t.Fatalf("ResolveAzExe() = %q, want %q", got, override)
	}
}

func writeFakeAz(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "az.cmd")
		script := "@echo off\r\n"
		script += "setlocal EnableDelayedExpansion\r\n"
		script += "set joined=\r\n"
		script += ":loop\r\n"
		script += "if \"%~1\"==\"\" goto done\r\n"
		script += "if defined joined (set joined=!joined! %~1) else (set joined=%~1)\r\n"
		script += "shift\r\n"
		script += "goto loop\r\n"
		script += ":done\r\n"
		script += "echo {\"argv\":\"!joined!\"}\r\n"
		if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
		return path
	}

	path := filepath.Join(dir, "az")
	script := "#!/bin/sh\nprintf '{\"argv\":\"%s\"}\\n' \"$*\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return path
}
