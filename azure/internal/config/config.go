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
	AppConfig      string
	Cosmos         string
	DataRoot       string
	TenantID       string
	SubscriptionID string
	TokenIssuer    string
	TokenAudience  string
	TokenSubject   string
	TokenKey       string
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
		AppConfig:      envOrDefault("TINYCLOUD_APPCONFIG_PORT", "4582"),
		Cosmos:         envOrDefault("TINYCLOUD_COSMOS_PORT", "4583"),
		DataRoot:       envOrDefault("TINYCLOUD_DATA_ROOT", defaultDataRoot()),
		TenantID:       envOrDefault("TINYCLOUD_TENANT_ID", defaultTenantID()),
		SubscriptionID: envOrDefault("TINYCLOUD_SUBSCRIPTION_ID", defaultSubscriptionID()),
		TokenIssuer:    envOrDefault("TINYCLOUD_TOKEN_ISSUER", ""),
		TokenAudience:  envOrDefault("TINYCLOUD_TOKEN_AUDIENCE", "https://management.azure.com/"),
		TokenSubject:   envOrDefault("TINYCLOUD_TOKEN_SUBJECT", "tinycloud-local-user"),
		TokenKey:       envOrDefault("TINYCLOUD_TOKEN_KEY", "tinycloud-dev-signing-key"),
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

func (c Config) ManagementTLSURL() string {
	return fmt.Sprintf("https://%s:%s", c.AdvertiseHost, c.ManagementTLS)
}

func (c Config) BlobURL() string {
	return fmt.Sprintf("http://%s:%s", c.AdvertiseHost, c.Blob)
}

func (c Config) QueueURL() string {
	return fmt.Sprintf("http://%s:%s", c.AdvertiseHost, c.Queue)
}

func (c Config) TableURL() string {
	return fmt.Sprintf("http://%s:%s", c.AdvertiseHost, c.Table)
}

func (c Config) KeyVaultURL() string {
	return fmt.Sprintf("http://%s:%s", c.AdvertiseHost, c.KeyVault)
}

func (c Config) ServiceBusURL() string {
	return fmt.Sprintf("http://%s:%s", c.AdvertiseHost, c.ServiceBus)
}

func (c Config) AppConfigURL() string {
	return fmt.Sprintf("http://%s:%s", c.AdvertiseHost, c.AppConfig)
}

func (c Config) CosmosURL() string {
	return fmt.Sprintf("http://%s:%s", c.AdvertiseHost, c.Cosmos)
}

func (c Config) OAuthTokenURL() string {
	return fmt.Sprintf("%s/oauth/token", c.ManagementHTTPURL())
}

func (c Config) ManagedIdentityURL() string {
	return fmt.Sprintf("%s/metadata/identity/oauth2/token", c.ManagementHTTPURL())
}

func (c Config) EffectiveTokenIssuer() string {
	if c.TokenIssuer != "" {
		return c.TokenIssuer
	}
	return c.OAuthTokenURL()
}

func (c Config) EndpointMap() map[string]string {
	return map[string]string{
		"management":      c.ManagementHTTPURL(),
		"managementHttps": c.ManagementTLSURL(),
		"metadata":        fmt.Sprintf("%s/metadata/endpoints", c.ManagementHTTPURL()),
		"identity":        fmt.Sprintf("%s/metadata/identity", c.ManagementHTTPURL()),
		"managedIdentity": c.ManagedIdentityURL(),
		"oauthToken":      c.OAuthTokenURL(),
		"tenants":         fmt.Sprintf("%s/tenants", c.ManagementHTTPURL()),
		"subscriptions":   fmt.Sprintf("%s/subscriptions", c.ManagementHTTPURL()),
		"blob":            c.BlobURL(),
		"queue":           c.QueueURL(),
		"table":           c.TableURL(),
		"keyVault":        c.KeyVaultURL(),
		"serviceBus":      c.ServiceBusURL(),
		"appConfig":       c.AppConfigURL(),
		"cosmos":          c.CosmosURL(),
	}
}

func defaultTenantID() string {
	return "00000000-0000-0000-0000-000000000001"
}

func defaultSubscriptionID() string {
	return "11111111-1111-1111-1111-111111111111"
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
