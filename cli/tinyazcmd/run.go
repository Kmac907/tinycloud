package tinyazcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"tinycloud-root/cli/tinyterraformcmd"
)

type envLoader func(cwd string, required []string) (map[string]string, error)
type azLookPath func(string) (string, error)

type accountOutput struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	User             accountUser     `json:"user"`
	TenantID         string          `json:"tenantId"`
	EnvironmentName  string          `json:"environmentName"`
	IsDefault        bool            `json:"isDefault"`
	ManagedByTenants []accountTenant `json:"managedByTenants,omitempty"`
}

type accountUser struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type accountTenant struct {
	TenantID string `json:"tenantId"`
}

type tokenOutput struct {
	AccessToken  string `json:"accessToken"`
	ExpiresOn    string `json:"expiresOn"`
	ExpiresUnix  int64  `json:"expires_on"`
	Subscription string `json:"subscription"`
	Tenant       string `json:"tenant"`
	TokenType    string `json:"tokenType"`
}

type tinyCloudTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func Main() {
	os.Exit(Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	code, err := RunE(args, stdin, stdout, stderr, os.Getwd, LoadTinyCloudTerraformEnv, time.Now, exec.LookPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
	}
	return code
}

func RunE(args []string, stdin io.Reader, stdout, stderr io.Writer, getwd func() (string, error), loadEnv envLoader, now func() time.Time, lookPath azLookPath) (int, error) {
	if len(args) == 0 {
		PrintUsage(stdout)
		return 2, errors.New("usage: tinyaz <az arguments>")
	}
	if isTinyCloudAccountFlow(args) {
		return runAccount(args[1:], stdout, getwd, loadEnv, now)
	}

	azExe, err := ResolveAzExe(lookPath)
	if err != nil {
		return 1, err
	}
	return tinyterraformcmd.RunCommand(azExe, args, stdin, stdout, stderr)
}

func runAccount(args []string, stdout io.Writer, getwd func() (string, error), loadEnv envLoader, now func() time.Time) (int, error) {
	if len(args) == 0 {
		return 2, errors.New("usage: tinyaz account <show|list|get-access-token>")
	}

	cwd, err := getwd()
	if err != nil {
		return 1, fmt.Errorf("resolve current directory: %w", err)
	}

	switch args[0] {
	case "show":
		values, err := loadEnv(cwd, []string{"ARM_SUBSCRIPTION_ID", "ARM_TENANT_ID"})
		if err != nil {
			return 1, err
		}
		return 0, writeJSON(stdout, accountFromEnv(values))
	case "list":
		values, err := loadEnv(cwd, []string{"ARM_SUBSCRIPTION_ID", "ARM_TENANT_ID"})
		if err != nil {
			return 1, err
		}
		return 0, writeJSON(stdout, []accountOutput{accountFromEnv(values)})
	case "get-access-token":
		values, err := loadEnv(cwd, []string{"ARM_SUBSCRIPTION_ID", "ARM_TENANT_ID", "TINY_OAUTH_TOKEN"})
		if err != nil {
			return 1, err
		}
		token, err := requestTinyCloudAccessToken(values["TINY_OAUTH_TOKEN"], tokenResource(args[1:]))
		if err != nil {
			return 1, err
		}
		expiresAt := now().Add(time.Hour).UTC()
		return 0, writeJSON(stdout, tokenOutput{
			AccessToken:  token.AccessToken,
			ExpiresOn:    expiresAt.Format("2006-01-02 15:04:05.000000"),
			ExpiresUnix:  expiresAt.Unix(),
			Subscription: values["ARM_SUBSCRIPTION_ID"],
			Tenant:       values["ARM_TENANT_ID"],
			TokenType:    "Bearer",
		})
	default:
		return 2, fmt.Errorf("unsupported az account command %q; supported commands: show, list, get-access-token", args[0])
	}
}

func isTinyCloudAccountFlow(args []string) bool {
	if len(args) < 2 || args[0] != "account" {
		return false
	}

	switch args[1] {
	case "show", "list":
		return supportsJSONOutputOnly(args[2:])
	case "get-access-token":
		return supportsTinyCloudTokenArgs(args[2:])
	default:
		return false
	}
}

func supportsJSONOutputOnly(args []string) bool {
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-o" || args[i] == "--output":
			if i+1 >= len(args) || !isJSONOutput(args[i+1]) {
				return false
			}
			i++
		case strings.HasPrefix(args[i], "--output="):
			if !isJSONOutput(strings.TrimPrefix(args[i], "--output=")) {
				return false
			}
		case strings.HasPrefix(args[i], "-o="):
			if !isJSONOutput(strings.TrimPrefix(args[i], "-o=")) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func supportsTinyCloudTokenArgs(args []string) bool {
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--resource" || args[i] == "--scope" || args[i] == "-o" || args[i] == "--output":
			if i+1 >= len(args) {
				return false
			}
			if (args[i] == "-o" || args[i] == "--output") && !isJSONOutput(args[i+1]) {
				return false
			}
			i++
		case strings.HasPrefix(args[i], "--resource="):
		case strings.HasPrefix(args[i], "--scope="):
		case strings.HasPrefix(args[i], "--output="):
			if !isJSONOutput(strings.TrimPrefix(args[i], "--output=")) {
				return false
			}
		case strings.HasPrefix(args[i], "-o="):
			if !isJSONOutput(strings.TrimPrefix(args[i], "-o=")) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func isJSONOutput(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "json", "jsonc":
		return true
	default:
		return false
	}
}

func accountFromEnv(values map[string]string) accountOutput {
	return accountOutput{
		ID:   values["ARM_SUBSCRIPTION_ID"],
		Name: "TinyCloud",
		User: accountUser{
			Name: "tinycloud",
			Type: "servicePrincipal",
		},
		TenantID:        values["ARM_TENANT_ID"],
		EnvironmentName: "AzureCloud",
		IsDefault:       true,
	}
}

func tokenResource(args []string) string {
	resource := "https://management.azure.com/"
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--resource" && i+1 < len(args):
			resource = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--resource="):
			resource = strings.TrimPrefix(args[i], "--resource=")
		case args[i] == "--scope" && i+1 < len(args):
			resource = strings.TrimSuffix(args[i+1], "/.default")
			i++
		case strings.HasPrefix(args[i], "--scope="):
			resource = strings.TrimSuffix(strings.TrimPrefix(args[i], "--scope="), "/.default")
		}
	}
	return resource
}

func requestTinyCloudAccessToken(tokenURL, resource string) (tinyCloudTokenResponse, error) {
	form := url.Values{}
	form.Set("resource", resource)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(tokenURL, form)
	if err != nil {
		return tinyCloudTokenResponse{}, fmt.Errorf("request TinyCloud access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return tinyCloudTokenResponse{}, fmt.Errorf("request TinyCloud access token: token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload tinyCloudTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return tinyCloudTokenResponse{}, fmt.Errorf("decode TinyCloud access token response: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return tinyCloudTokenResponse{}, errors.New("decode TinyCloud access token response: missing access_token")
	}
	return payload, nil
}

func LoadTinyCloudTerraformEnv(cwd string, required []string) (map[string]string, error) {
	repoRoot, err := tinyterraformcmd.ResolveTinyCloudRepoRoot(cwd)
	if err != nil {
		return nil, err
	}

	runtimeRoot, err := os.MkdirTemp("", "tinyaz-runtime-*")
	if err != nil {
		return nil, fmt.Errorf("create tinyaz runtime root: %w", err)
	}
	defer os.RemoveAll(runtimeRoot)

	tinycloudExe, err := tinyterraformcmd.BuildTinyCloudExe(repoRoot, runtimeRoot)
	if err != nil {
		return nil, err
	}

	var envStdout bytes.Buffer
	var envStderr bytes.Buffer
	code, err := tinyterraformcmd.RunCommand(tinycloudExe, []string{"env", "terraform"}, nil, &envStdout, &envStderr)
	if err != nil {
		return nil, fmt.Errorf("load tinycloud terraform environment: %w", err)
	}
	if code != 0 {
		detail := strings.TrimSpace(envStderr.String())
		if detail == "" {
			detail = strings.TrimSpace(envStdout.String())
		}
		if detail != "" {
			return nil, fmt.Errorf("load tinycloud terraform environment: tinycloud env terraform exited %d: %s", code, detail)
		}
		return nil, fmt.Errorf("load tinycloud terraform environment: tinycloud env terraform exited %d", code)
	}

	values, err := tinyterraformcmd.ParseTerraformEnv(envStdout.String(), required)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func ResolveAzExe(lookPath azLookPath) (string, error) {
	if override := strings.TrimSpace(os.Getenv("TINYAZ_EXE")); override != "" {
		return override, nil
	}
	if lookPath == nil {
		return "", errors.New("Azure CLI was not found. Install Azure CLI or set TINYAZ_EXE")
	}
	path, err := lookPath("az")
	if err != nil {
		return "", errors.New("Azure CLI was not found. Install Azure CLI or set TINYAZ_EXE")
	}
	return path, nil
}

func writeJSON(w io.Writer, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(body))
	return err
}

func PrintUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `tinyaz commands:
  <any az command>
  account show
  account list
  account get-access-token [--resource <resource> | --scope <scope>]

Current TinyCloud-routed subset:
  account show
  account list
  account get-access-token

Other commands pass through to the real az CLI.`)
}
