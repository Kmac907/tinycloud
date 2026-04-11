package tinycloudcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tinycloud/runtime/tinycloudconfig"
)

const (
	defaultDockerImage      = "tinycloud-azure"
	containerDataRoot       = "/var/lib/tinycloud"
	containerLabelManagedBy = "tinycloud.cli.managed=true"
)

type dockerRuntime struct {
	Image         string   `json:"image"`
	ContainerName string   `json:"containerName"`
	ContainerID   string   `json:"containerId,omitempty"`
	Network       string   `json:"network,omitempty"`
	Env           []string `json:"env,omitempty"`
	Publish       []string `json:"publish,omitempty"`
	Volumes       []string `json:"volumes,omitempty"`
}

type dockerContainerState struct {
	ID      string
	Name    string
	Image   string
	Status  string
	Running bool
}

func dockerAvailable() bool {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	return cmd.Run() == nil
}

func ensureDockerImage(repoRoot, image string) error {
	inspect := exec.Command("docker", "image", "inspect", image)
	if err := inspect.Run(); err == nil {
		return nil
	}

	build := exec.Command("docker", "build", "-t", image, ".")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("build docker image %q: %w: %s", image, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func dockerContainerRunning(name string) (bool, error) {
	state, ok, err := inspectDockerContainer(name)
	if err != nil || !ok {
		return false, err
	}
	return state.Running, nil
}

func inspectDockerContainer(name string) (dockerContainerState, bool, error) {
	if strings.TrimSpace(name) == "" {
		return dockerContainerState{}, false, nil
	}
	cmd := exec.Command("docker", "inspect", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.ToLower(string(output))
		if strings.Contains(text, "no such object") {
			return dockerContainerState{}, false, nil
		}
		return dockerContainerState{}, false, fmt.Errorf("inspect docker container %q: %w: %s", name, err, strings.TrimSpace(string(output)))
	}

	var body []struct {
		ID     string `json:"Id"`
		Name   string `json:"Name"`
		Config struct {
			Image string `json:"Image"`
		} `json:"Config"`
		State struct {
			Status  string `json:"Status"`
			Running bool   `json:"Running"`
		} `json:"State"`
	}
	if err := json.Unmarshal(output, &body); err != nil {
		return dockerContainerState{}, false, fmt.Errorf("decode docker inspect: %w", err)
	}
	if len(body) == 0 {
		return dockerContainerState{}, false, nil
	}

	return dockerContainerState{
		ID:      body[0].ID,
		Name:    strings.TrimPrefix(body[0].Name, "/"),
		Image:   body[0].Config.Image,
		Status:  body[0].State.Status,
		Running: body[0].State.Running,
	}, true, nil
}

func removeDockerContainer(name string) error {
	state, ok, err := inspectDockerContainer(name)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	cmd := exec.Command("docker", "rm", "-f", state.Name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("remove docker container %q: %w: %s", state.Name, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func dockerLogs(name string, follow bool, stdout io.Writer) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, name)
	cmd := exec.Command("docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	return cmd.Run()
}

func defaultDockerPublishes(cfg tinycloudconfig.Config) []string {
	values := []string{}
	if cfg.ServiceEnabled(tinycloudconfig.ServiceManagement) {
		values = append(values,
			fmt.Sprintf("%s:%s", cfg.ManagementHTTP, cfg.ManagementHTTP),
			fmt.Sprintf("%s:%s", cfg.ManagementTLS, cfg.ManagementTLS),
		)
	}
	for _, service := range cfg.EnabledServices() {
		switch service {
		case tinycloudconfig.ServiceBlob:
			values = append(values, fmt.Sprintf("%s:%s", cfg.Blob, cfg.Blob))
		case tinycloudconfig.ServiceQueue:
			values = append(values, fmt.Sprintf("%s:%s", cfg.Queue, cfg.Queue))
		case tinycloudconfig.ServiceTable:
			values = append(values, fmt.Sprintf("%s:%s", cfg.Table, cfg.Table))
		case tinycloudconfig.ServiceKeyVault:
			values = append(values, fmt.Sprintf("%s:%s", cfg.KeyVault, cfg.KeyVault))
		case tinycloudconfig.ServiceServiceBus:
			values = append(values, fmt.Sprintf("%s:%s", cfg.ServiceBus, cfg.ServiceBus))
		case tinycloudconfig.ServiceAppConfig:
			values = append(values, fmt.Sprintf("%s:%s", cfg.AppConfig, cfg.AppConfig))
		case tinycloudconfig.ServiceCosmos:
			values = append(values, fmt.Sprintf("%s:%s", cfg.Cosmos, cfg.Cosmos))
		case tinycloudconfig.ServiceDNS:
			values = append(values, fmt.Sprintf("%s:%s/udp", cfg.DNS, cfg.DNS))
		case tinycloudconfig.ServiceEventHubs:
			values = append(values, fmt.Sprintf("%s:%s", cfg.EventHubs, cfg.EventHubs))
		}
	}
	return values
}

func containerEnvValues(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		if strings.HasPrefix(key, "TINYCLOUD_") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		value := env[key]
		if key == "TINYCLOUD_DATA_ROOT" {
			value = containerDataRoot
		}
		values = append(values, key+"="+value)
	}
	return values
}

func dockerContainerName(runtimeRoot string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(runtimeRoot))
	return fmt.Sprintf("tinycloud-%08x", hash.Sum32())
}

func dockerRunArgs(containerName, image string, cfg tinycloudconfig.Config, env map[string]string, spec dockerRuntime) []string {
	args := []string{
		"run",
		"-d",
		"--name", containerName,
		"--label", containerLabelManagedBy,
	}
	if spec.Network != "" {
		args = append(args, "--network", spec.Network)
	}

	hostDataRoot := cfg.DataRoot
	if !filepath.IsAbs(hostDataRoot) {
		hostDataRoot = filepath.Clean(hostDataRoot)
	}
	args = append(args, "-v", hostDataRoot+":"+containerDataRoot)

	for _, value := range containerEnvValues(env) {
		args = append(args, "-e", value)
	}
	for _, value := range spec.Env {
		args = append(args, "-e", value)
	}
	for _, value := range defaultDockerPublishes(cfg) {
		args = append(args, "-p", value)
	}
	for _, value := range spec.Publish {
		args = append(args, "-p", value)
	}
	for _, value := range spec.Volumes {
		args = append(args, "-v", value)
	}
	args = append(args, image)
	return args
}

func startDockerRuntime(ctx cliContext, spec dockerRuntime, detached, jsonOutput bool, stdout io.Writer, showBanner bool) (int, error) {
	ui := newTerminalUI(stdout)
	if err := os.MkdirAll(ctx.config.DataRoot, 0o755); err != nil {
		return 1, fmt.Errorf("create TinyCloud data root: %w", err)
	}
	if spec.Image == "" {
		spec.Image = defaultDockerImage
	}
	if spec.ContainerName == "" {
		spec.ContainerName = dockerContainerName(ctx.runtimeRoot)
	}

	if err := ensureDockerImage(ctx.repoRoot, spec.Image); err != nil {
		return 1, err
	}
	if err := removeDockerContainer(spec.ContainerName); err != nil {
		return 1, err
	}

	cmd := exec.Command("docker", dockerRunArgs(spec.ContainerName, spec.Image, ctx.config, ctx.env, spec)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 1, fmt.Errorf("start TinyCloud docker runtime: %w: %s", err, strings.TrimSpace(string(output)))
	}
	spec.ContainerID = strings.TrimSpace(string(output))

	record := runtimeRecord{
		Backend:     "docker",
		StartedAt:   nowUTC(),
		RepoRoot:    ctx.repoRoot,
		RuntimeRoot: ctx.runtimeRoot,
		Detached:    true,
		Env:         copyMap(ctx.env),
		Config:      ctx.config,
		Docker:      &spec,
	}
	if err := saveRuntimeRecord(ctx.runtimeRoot, record); err != nil {
		_ = removeDockerContainer(spec.ContainerName)
		return 1, err
	}
	if err := waitForHealthy(ctx.config, defaultWaitTimeout); err != nil {
		_ = removeDockerContainer(spec.ContainerName)
		_ = removeRuntimeRecord(ctx.runtimeRoot)
		return 1, fmt.Errorf("wait for docker runtime: %w", err)
	}

	summary := map[string]any{
		"status":       "running",
		"runtimeId":    firstNonEmpty(spec.ContainerID, spec.ContainerName),
		"backend":      "docker",
		"containerId":  spec.ContainerID,
		"container":    spec.ContainerName,
		"image":        spec.Image,
		"services":     ctx.config.EnabledServices(),
		"endpoints":    ctx.config.EndpointMap(),
		"nextCommands": []string{"tinycloud status runtime", "tinycloud logs -f", "tinycloud stop"},
	}
	if ctx.config.ServiceEnabled(tinycloudconfig.ServiceManagement) {
		summary["management"] = ctx.config.ManagementHTTPURL()
	}
	if jsonOutput {
		if err := formatJSON(stdout, summary); err != nil {
			return 1, err
		}
	} else if detached {
		output := renderDetachedStartOutput(ui, showBanner, "docker", startSummary{
			RuntimeID:  firstNonEmpty(spec.ContainerID, spec.ContainerName),
			Backend:    "docker",
			Container:  spec.ContainerName,
			Image:      spec.Image,
			Services:   joinServices(ctx.config.EnabledServices()),
			Management: managementValue(ctx.config),
			Endpoints:  ctx.config.EndpointMap(),
		}, []string{ui.success("build image"), ui.success("create container"), ui.success("wait for health")})
		if err := writeString(stdout, output); err != nil {
			return 1, err
		}
	}

	if detached {
		return 0, nil
	}
	if !jsonOutput {
		if err := writeString(stdout, renderAttachedStartPrelude(ui, showBanner, "docker", startSummary{
			RuntimeID:  firstNonEmpty(spec.ContainerID, spec.ContainerName),
			Backend:    "docker",
			Container:  spec.ContainerName,
			Image:      spec.Image,
			Services:   joinServices(ctx.config.EnabledServices()),
			Management: managementValue(ctx.config),
		}, []string{ui.success("build image"), ui.success("create container"), ui.success("wait for health"), ui.progress("stream logs")})); err != nil {
			return 1, err
		}
	}
	return 0, dockerLogs(spec.ContainerName, true, stdout)
}

func stopDockerRuntime(record runtimeRecord) error {
	if record.Docker == nil {
		return errors.New("docker runtime metadata is missing")
	}
	return removeDockerContainer(record.Docker.ContainerName)
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
