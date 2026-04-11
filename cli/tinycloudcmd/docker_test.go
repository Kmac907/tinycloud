package tinycloudcmd

import (
	"reflect"
	"testing"

	"tinycloud/runtime/tinycloudconfig"
)

func TestDefaultDockerPublishesOnlyEnabledServices(t *testing.T) {
	t.Parallel()

	cfg := tinycloudconfig.FromMap(map[string]string{
		"TINYCLOUD_SERVICES":        "management,messaging",
		"TINYCLOUD_MGMT_HTTP_PORT":  "5566",
		"TINYCLOUD_MGMT_HTTPS_PORT": "5567",
		"TINYCLOUD_SERVICEBUS_PORT": "5581",
		"TINYCLOUD_EVENTHUBS_PORT":  "5585",
	})

	got := defaultDockerPublishes(cfg)
	want := []string{
		"5566:5566",
		"5567:5567",
		"5581:5581",
		"5585:5585",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("defaultDockerPublishes() = %#v, want %#v", got, want)
	}
}

func TestContainerEnvValuesUseContainerDataRoot(t *testing.T) {
	t.Parallel()

	got := containerEnvValues(map[string]string{
		"TINYCLOUD_DATA_ROOT":      `C:\temp\tinycloud-data`,
		"TINYCLOUD_SERVICES":       "management",
		"TINYCLOUD_MGMT_HTTP_PORT": "4566",
		"GOCACHE":                  `C:\temp\.gocache`,
	})

	want := []string{
		"TINYCLOUD_DATA_ROOT=/var/lib/tinycloud",
		"TINYCLOUD_MGMT_HTTP_PORT=4566",
		"TINYCLOUD_SERVICES=management",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("containerEnvValues() = %#v, want %#v", got, want)
	}
}

func TestResolveRuntimeBackendPrefersExplicitValues(t *testing.T) {
	t.Parallel()

	if got := resolveRuntimeBackend("process", map[string]string{"TINYCLOUD_BACKEND": "docker"}); got != "process" {
		t.Fatalf("resolveRuntimeBackend(flag) = %q, want %q", got, "process")
	}
	if got := resolveRuntimeBackend("", map[string]string{"TINYCLOUD_BACKEND": "docker"}); got != "docker" {
		t.Fatalf("resolveRuntimeBackend(env) = %q, want %q", got, "docker")
	}
}
