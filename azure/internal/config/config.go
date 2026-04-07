package config

import (
	"fmt"
	"os"
	"runtime"
)

type Config struct {
	ListenHost     string
	AdvertiseHost  string
	ManagementHTTP string
	ManagementTLS  string
	Blob           string
	Queue          string
	Table          string
	KeyVault       string
	ServiceBus     string
	DataRoot       string
}

func FromEnv() Config {
	advertiseHost := envOrDefault("TINYCLOUD_ADVERTISE_HOST", defaultAdvertiseHost())
	host := envOrDefault("TINYCLOUD_HOST", "")
	if host != "" {
		advertiseHost = host
	}

	return Config{
		ListenHost:     envOrDefault("TINYCLOUD_LISTEN_HOST", defaultListenHost()),
		AdvertiseHost:  advertiseHost,
		ManagementHTTP: envOrDefault("TINYCLOUD_MGMT_HTTP_PORT", "4566"),
		ManagementTLS:  envOrDefault("TINYCLOUD_MGMT_HTTPS_PORT", "4567"),
		Blob:           envOrDefault("TINYCLOUD_BLOB_PORT", "4577"),
		Queue:          envOrDefault("TINYCLOUD_QUEUE_PORT", "4578"),
		Table:          envOrDefault("TINYCLOUD_TABLE_PORT", "4579"),
		KeyVault:       envOrDefault("TINYCLOUD_KEYVAULT_PORT", "4580"),
		ServiceBus:     envOrDefault("TINYCLOUD_SERVICEBUS_PORT", "4581"),
		DataRoot:       envOrDefault("TINYCLOUD_DATA_ROOT", defaultDataRoot()),
	}
}

func (c Config) ManagementAddr() string {
	return fmt.Sprintf("%s:%s", c.ListenHost, c.ManagementHTTP)
}

func (c Config) ManagementHost() string {
	return c.AdvertiseHost
}

func (c Config) ManagementHTTPURL() string {
	return fmt.Sprintf("http://%s:%s", c.AdvertiseHost, c.ManagementHTTP)
}

func (c Config) EndpointMap() map[string]string {
	return map[string]string{
		"management": c.ManagementHTTPURL(),
	}
}

func defaultDataRoot() string {
	if runtime.GOOS == "windows" {
		return ".\\data"
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return fmt.Sprintf("%s/.tinycloud/data", home)
	}
	return "./data"
}

func defaultListenHost() string {
	if runtime.GOOS == "windows" {
		return "127.0.0.1"
	}
	return "0.0.0.0"
}

func defaultAdvertiseHost() string {
	return "127.0.0.1"
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
