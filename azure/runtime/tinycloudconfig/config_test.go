package tinycloudconfig

import "testing"

func TestParseServiceSelectionExpandsFamilies(t *testing.T) {
	t.Parallel()

	selection := ParseServiceSelection("management, storage, messaging")

	for _, service := range []Service{
		ServiceManagement,
		ServiceBlob,
		ServiceQueue,
		ServiceTable,
		ServiceServiceBus,
		ServiceEventHubs,
	} {
		if !selection.Has(service) {
			t.Fatalf("selection missing %q", service)
		}
	}
	if selection.Has(ServiceDNS) {
		t.Fatalf("selection unexpectedly enabled %q", ServiceDNS)
	}
}

func TestParseServiceSelectionReportsInvalidEntries(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ListenHost:     "127.0.0.1",
		AdvertiseHost:  "127.0.0.1",
		ManagementHTTP: "4566",
		ManagementTLS:  "4567",
		Blob:           "4577",
		Queue:          "4578",
		Table:          "4579",
		KeyVault:       "4580",
		ServiceBus:     "4581",
		AppConfig:      "4582",
		Cosmos:         "4583",
		DNS:            "4584",
		EventHubs:      "4585",
		Services:       ParseServiceSelection("management,unknown-service"),
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid service selection")
	}
}

func TestEndpointMapFiltersDisabledServices(t *testing.T) {
	t.Parallel()

	cfg := Config{
		AdvertiseHost:  "127.0.0.1",
		ManagementHTTP: "4566",
		ManagementTLS:  "4567",
		Blob:           "4577",
		Queue:          "4578",
		Table:          "4579",
		KeyVault:       "4580",
		ServiceBus:     "4581",
		AppConfig:      "4582",
		Cosmos:         "4583",
		DNS:            "4584",
		EventHubs:      "4585",
		Services:       ParseServiceSelection("management,blob"),
	}

	endpoints := cfg.EndpointMap()
	if endpoints["management"] == "" || endpoints["blob"] == "" {
		t.Fatalf("EndpointMap() missing enabled services: %#v", endpoints)
	}
	if _, ok := endpoints["queue"]; ok {
		t.Fatalf("EndpointMap() unexpectedly included disabled queue service: %#v", endpoints)
	}
	if _, ok := endpoints["eventHubs"]; ok {
		t.Fatalf("EndpointMap() unexpectedly included disabled event hubs service: %#v", endpoints)
	}
}
