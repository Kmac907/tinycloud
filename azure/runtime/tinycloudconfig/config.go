package tinycloudconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type Service string

const (
	ServiceManagement Service = "management"
	ServiceBlob       Service = "blob"
	ServiceQueue      Service = "queue"
	ServiceTable      Service = "table"
	ServiceKeyVault   Service = "keyVault"
	ServiceServiceBus Service = "serviceBus"
	ServiceAppConfig  Service = "appConfig"
	ServiceCosmos     Service = "cosmos"
	ServiceDNS        Service = "dns"
	ServiceEventHubs  Service = "eventHubs"
)

type ServiceDescriptor struct {
	Name     Service `json:"name"`
	Family   string  `json:"family"`
	Enabled  bool    `json:"enabled"`
	Endpoint string  `json:"endpoint,omitempty"`
}

type ServiceSelection struct {
	raw     string
	enabled map[Service]struct{}
	invalid []string
}

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
	DNS            string
	EventHubs      string
	DataRoot       string
	TenantID       string
	SubscriptionID string
	TokenIssuer    string
	TokenAudience  string
	TokenSubject   string
	TokenKey       string
	Services       ServiceSelection
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
		DNS:            envOrDefault("TINYCLOUD_DNS_PORT", "4584"),
		EventHubs:      envOrDefault("TINYCLOUD_EVENTHUBS_PORT", "4585"),
		DataRoot:       envOrDefault("TINYCLOUD_DATA_ROOT", defaultDataRoot()),
		TenantID:       envOrDefault("TINYCLOUD_TENANT_ID", defaultTenantID()),
		SubscriptionID: envOrDefault("TINYCLOUD_SUBSCRIPTION_ID", defaultSubscriptionID()),
		TokenIssuer:    envOrDefault("TINYCLOUD_TOKEN_ISSUER", ""),
		TokenAudience:  envOrDefault("TINYCLOUD_TOKEN_AUDIENCE", "https://management.azure.com/"),
		TokenSubject:   envOrDefault("TINYCLOUD_TOKEN_SUBJECT", "tinycloud-local-user"),
		TokenKey:       envOrDefault("TINYCLOUD_TOKEN_KEY", "tinycloud-dev-signing-key"),
		Services:       ParseServiceSelection(os.Getenv("TINYCLOUD_SERVICES")),
	}
}

func (c Config) Validate() error {
	if err := c.Services.Validate(); err != nil {
		return err
	}

	addresses := map[string]string{}
	for _, service := range c.EnabledServices() {
		for _, address := range c.listenerAddresses(service) {
			if address == "" {
				continue
			}
			if existing, ok := addresses[address]; ok {
				return fmt.Errorf("service %q shares listener address %q with %q", service, address, existing)
			}
			addresses[address] = string(service)
		}
	}

	return nil
}

func (c Config) ServiceEnabled(service Service) bool {
	return c.Services.Has(service)
}

func (c Config) EnabledServices() []Service {
	return c.Services.Names()
}

func (c Config) DisabledServices() []Service {
	disabled := make([]Service, 0, len(allServices))
	for _, service := range allServices {
		if !c.ServiceEnabled(service) {
			disabled = append(disabled, service)
		}
	}
	return disabled
}

func (c Config) ServiceCatalog() []ServiceDescriptor {
	services := make([]ServiceDescriptor, 0, len(allServices))
	for _, service := range allServices {
		services = append(services, ServiceDescriptor{
			Name:     service,
			Family:   serviceFamily(service),
			Enabled:  c.ServiceEnabled(service),
			Endpoint: c.serviceEndpoint(service),
		})
	}
	return services
}

func (c Config) ManagementAddr() string {
	return fmt.Sprintf("%s:%s", c.ListenHost, c.ManagementHTTP)
}

func (c Config) ManagementTLSAddr() string {
	return fmt.Sprintf("%s:%s", c.ListenHost, c.ManagementTLS)
}

func (c Config) ManagementHost() string {
	return c.AdvertiseHost
}

func (c Config) ManagementTLSHost() string {
	return fmt.Sprintf("%s:%s", c.AdvertiseHost, c.ManagementTLS)
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

func (c Config) DNSAddress() string {
	return fmt.Sprintf("%s:%s", c.AdvertiseHost, c.DNS)
}

func (c Config) DNSURL() string {
	return fmt.Sprintf("udp://%s", c.DNSAddress())
}

func (c Config) EventHubsURL() string {
	return fmt.Sprintf("http://%s:%s", c.AdvertiseHost, c.EventHubs)
}

func (c Config) OAuthTokenURL() string {
	return fmt.Sprintf("%s/oauth/token", c.ManagementHTTPURL())
}

func (c Config) ManagedIdentityURL() string {
	return fmt.Sprintf("%s/metadata/identity/oauth2/token", c.ManagementHTTPURL())
}

func (c Config) ManagementTLSCertPath() string {
	return filepath.Join(c.DataRoot, "tls", "management.crt")
}

func (c Config) ManagementTLSKeyPath() string {
	return filepath.Join(c.DataRoot, "tls", "management.key")
}

func (c Config) EffectiveTokenIssuer() string {
	if c.TokenIssuer != "" {
		return c.TokenIssuer
	}
	return c.OAuthTokenURL()
}

func (c Config) EndpointMap() map[string]string {
	endpoints := map[string]string{}
	if c.ServiceEnabled(ServiceManagement) {
		endpoints["management"] = c.ManagementHTTPURL()
		endpoints["managementHttps"] = c.ManagementTLSURL()
		endpoints["metadata"] = fmt.Sprintf("%s/metadata/endpoints", c.ManagementHTTPURL())
		endpoints["identity"] = fmt.Sprintf("%s/metadata/identity", c.ManagementHTTPURL())
		endpoints["managedIdentity"] = c.ManagedIdentityURL()
		endpoints["oauthToken"] = c.OAuthTokenURL()
		endpoints["tenants"] = fmt.Sprintf("%s/tenants", c.ManagementHTTPURL())
		endpoints["subscriptions"] = fmt.Sprintf("%s/subscriptions", c.ManagementHTTPURL())
	}
	for _, descriptor := range c.ServiceCatalog() {
		if descriptor.Name == ServiceManagement || !descriptor.Enabled || descriptor.Endpoint == "" {
			continue
		}
		endpoints[string(descriptor.Name)] = descriptor.Endpoint
	}
	return endpoints
}

func (c Config) serviceEndpoint(service Service) string {
	switch service {
	case ServiceManagement:
		return c.ManagementHTTPURL()
	case ServiceBlob:
		return c.BlobURL()
	case ServiceQueue:
		return c.QueueURL()
	case ServiceTable:
		return c.TableURL()
	case ServiceKeyVault:
		return c.KeyVaultURL()
	case ServiceServiceBus:
		return c.ServiceBusURL()
	case ServiceAppConfig:
		return c.AppConfigURL()
	case ServiceCosmos:
		return c.CosmosURL()
	case ServiceDNS:
		return c.DNSURL()
	case ServiceEventHubs:
		return c.EventHubsURL()
	default:
		return ""
	}
}

func (c Config) listenerAddresses(service Service) []string {
	switch service {
	case ServiceManagement:
		return []string{c.ManagementAddr(), c.ManagementTLSAddr()}
	case ServiceBlob:
		return []string{fmt.Sprintf("%s:%s", c.ListenHost, c.Blob)}
	case ServiceQueue:
		return []string{fmt.Sprintf("%s:%s", c.ListenHost, c.Queue)}
	case ServiceTable:
		return []string{fmt.Sprintf("%s:%s", c.ListenHost, c.Table)}
	case ServiceKeyVault:
		return []string{fmt.Sprintf("%s:%s", c.ListenHost, c.KeyVault)}
	case ServiceServiceBus:
		return []string{fmt.Sprintf("%s:%s", c.ListenHost, c.ServiceBus)}
	case ServiceAppConfig:
		return []string{fmt.Sprintf("%s:%s", c.ListenHost, c.AppConfig)}
	case ServiceCosmos:
		return []string{fmt.Sprintf("%s:%s", c.ListenHost, c.Cosmos)}
	case ServiceDNS:
		return []string{fmt.Sprintf("udp://%s:%s", c.ListenHost, c.DNS)}
	case ServiceEventHubs:
		return []string{fmt.Sprintf("%s:%s", c.ListenHost, c.EventHubs)}
	default:
		return nil
	}
}

func ParseServiceSelection(raw string) ServiceSelection {
	selection := ServiceSelection{raw: raw, enabled: map[Service]struct{}{}}
	if strings.TrimSpace(raw) == "" {
		for _, service := range allServices {
			selection.enabled[service] = struct{}{}
		}
		return selection
	}

	for _, token := range strings.Split(raw, ",") {
		normalized := normalizeServiceToken(token)
		if normalized == "" {
			continue
		}

		switch normalized {
		case "all":
			for _, service := range allServices {
				selection.enabled[service] = struct{}{}
			}
			continue
		case "none":
			selection.enabled = map[Service]struct{}{}
			continue
		}

		services, ok := serviceAliases[normalized]
		if !ok {
			selection.invalid = append(selection.invalid, strings.TrimSpace(token))
			continue
		}
		for _, service := range services {
			selection.enabled[service] = struct{}{}
		}
	}

	return selection
}

func (s ServiceSelection) Raw() string {
	return s.raw
}

func (s ServiceSelection) Names() []Service {
	names := make([]Service, 0, len(allServices))
	for _, service := range allServices {
		if s.Has(service) {
			names = append(names, service)
		}
	}
	return names
}

func (s ServiceSelection) Has(service Service) bool {
	_, ok := s.enabled[service]
	return ok
}

func (s ServiceSelection) Invalid() []string {
	if len(s.invalid) == 0 {
		return nil
	}
	values := append([]string(nil), s.invalid...)
	sort.Strings(values)
	return values
}

func (s ServiceSelection) Validate() error {
	if len(s.invalid) == 0 {
		return nil
	}
	return fmt.Errorf("unknown TINYCLOUD_SERVICES entries: %s", strings.Join(s.Invalid(), ", "))
}

var allServices = []Service{
	ServiceManagement,
	ServiceBlob,
	ServiceQueue,
	ServiceTable,
	ServiceKeyVault,
	ServiceServiceBus,
	ServiceAppConfig,
	ServiceCosmos,
	ServiceDNS,
	ServiceEventHubs,
}

var serviceAliases = map[string][]Service{
	"management":     {ServiceManagement},
	"control":        {ServiceManagement},
	"controlplane":   {ServiceManagement},
	"blob":           {ServiceBlob},
	"queue":          {ServiceQueue},
	"table":          {ServiceTable},
	"storage":        {ServiceBlob, ServiceQueue, ServiceTable},
	"keyvault":       {ServiceKeyVault},
	"secrets":        {ServiceKeyVault},
	"appconfig":      {ServiceAppConfig},
	"config":         {ServiceAppConfig},
	"secretsconfig":  {ServiceKeyVault, ServiceAppConfig},
	"cosmos":         {ServiceCosmos},
	"data":           {ServiceCosmos},
	"servicebus":     {ServiceServiceBus},
	"eventhubs":      {ServiceEventHubs},
	"messaging":      {ServiceServiceBus, ServiceEventHubs},
	"messagingevent": {ServiceServiceBus, ServiceEventHubs},
	"networking":     {ServiceDNS},
	"dns":            {ServiceDNS},
}

func serviceFamily(service Service) string {
	switch service {
	case ServiceManagement:
		return "control-plane"
	case ServiceBlob, ServiceQueue, ServiceTable:
		return "storage"
	case ServiceKeyVault, ServiceAppConfig:
		return "secrets-config"
	case ServiceCosmos:
		return "data"
	case ServiceServiceBus, ServiceEventHubs:
		return "messaging-event"
	case ServiceDNS:
		return "networking"
	default:
		return "unknown"
	}
}

func normalizeServiceToken(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("-", "", "_", "", " ", "")
	return replacer.Replace(normalized)
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

var errNoServices = errors.New("no TinyCloud services are enabled")

func (c Config) RequireServices() error {
	if len(c.EnabledServices()) == 0 {
		return errNoServices
	}
	return nil
}
