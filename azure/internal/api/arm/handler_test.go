package arm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tinycloud/internal/config"
	"tinycloud/internal/httpx"
	"tinycloud/internal/state"
)

func TestListSubscriptionsReturnsBootstrapRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions?api-version=2024-01-01", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(body.Value) != 1 {
		t.Fatalf("len(value) = %d, want %d", len(body.Value), 1)
	}
}

func TestListTenantsReturnsBootstrapRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/tenants?api-version=2024-01-01", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(body.Value) != 1 {
		t.Fatalf("len(value) = %d, want %d", len(body.Value), 1)
	}
	if body.Value[0]["id"] == "" || body.Value[0]["tenantId"] == "" {
		t.Fatalf("tenant response = %#v, want non-empty id and tenantId", body.Value[0])
	}
}

func TestListProvidersReturnsBootstrapProvider(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/providers?api-version=2024-01-01", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(body.Value) != 4 {
		t.Fatalf("len(value) = %d, want %d", len(body.Value), 4)
	}
}

func TestGetProviderReturnsBootstrapProvider(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/providers/Microsoft.Storage?api-version=2024-01-01", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["namespace"] != "Microsoft.Storage" {
		t.Fatalf("namespace = %v, want %q", body["namespace"], "Microsoft.Storage")
	}
}

func TestRegisterProviderUpdatesState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/test-sub/providers/Microsoft.Custom/register?api-version=2024-01-01", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	provider, err := store.GetProvider("Microsoft.Custom")
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}
	if provider.RegistrationState != "Registered" {
		t.Fatalf("RegistrationState = %q, want %q", provider.RegistrationState, "Registered")
	}
}

func TestResourceGroupCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	createReq := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one?api-version=2024-01-01", strings.NewReader(`{"location":"westus2","tags":{"env":"test"}}`))
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusAccepted)
	}
	if createRec.Header().Get("Azure-AsyncOperation") == "" {
		t.Fatal("Azure-AsyncOperation header is empty")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups?api-version=2024-01-01", nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}

	var listBody struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(listBody.Value) != 1 {
		t.Fatalf("len(value) = %d, want %d", len(listBody.Value), 1)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one?api-version=2024-01-01", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/subscriptions/test-sub/resourceGroups/rg-one?api-version=2024-01-01", nil)
	deleteRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusAccepted {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusAccepted)
	}
	if deleteRec.Header().Get("Azure-AsyncOperation") == "" {
		t.Fatal("delete Azure-AsyncOperation header is empty")
	}
}

func TestStorageAccountCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertResourceGroup("test-sub", "rg-one", "westus2", "", nil); err != nil {
		t.Fatalf("UpsertResourceGroup() error = %v", err)
	}

	cfg := config.FromEnv()
	mux := http.NewServeMux()
	NewHandler(store, cfg).Register(mux)

	createReq := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Storage/storageAccounts/storeone?api-version=2024-01-01", strings.NewReader(`{"location":"westus2","kind":"StorageV2","sku":{"name":"Standard_LRS"},"tags":{"env":"test"}}`))
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusAccepted)
	}
	if createRec.Header().Get("Azure-AsyncOperation") == "" {
		t.Fatal("Azure-AsyncOperation header is empty")
	}

	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal() create error = %v", err)
	}
	properties, _ := created["properties"].(map[string]any)
	primaryEndpoints, _ := properties["primaryEndpoints"].(map[string]any)
	if primaryEndpoints["blob"] != cfg.BlobURL()+"/storeone" {
		t.Fatalf("primary blob endpoint = %v, want %q", primaryEndpoints["blob"], cfg.BlobURL()+"/storeone")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Storage/storageAccounts?api-version=2024-01-01", nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Storage/storageAccounts/storeone?api-version=2024-01-01", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Storage/storageAccounts/storeone?api-version=2024-01-01", nil)
	deleteRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusAccepted {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusAccepted)
	}
}

func TestPutStorageAccountRequiresExistingResourceGroup(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/missing/providers/Microsoft.Storage/storageAccounts/storeone?api-version=2024-01-01", strings.NewReader(`{"location":"westus2"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestKeyVaultCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertResourceGroup("test-sub", "rg-one", "westus2", "", nil); err != nil {
		t.Fatalf("UpsertResourceGroup() error = %v", err)
	}

	cfg := config.FromEnv()
	mux := http.NewServeMux()
	NewHandler(store, cfg).Register(mux)

	createReq := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.KeyVault/vaults/vaultone?api-version=2024-01-01", strings.NewReader(`{"location":"westus2","properties":{"tenantId":"tenant-123","sku":{"name":"standard"}},"tags":{"env":"test"}}`))
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusAccepted)
	}
	if createRec.Header().Get("Azure-AsyncOperation") == "" {
		t.Fatal("Azure-AsyncOperation header is empty")
	}

	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal() create error = %v", err)
	}
	properties, _ := created["properties"].(map[string]any)
	if properties["vaultUri"] != cfg.KeyVaultURL()+"/vaultone" {
		t.Fatalf("vaultUri = %v, want %q", properties["vaultUri"], cfg.KeyVaultURL()+"/vaultone")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.KeyVault/vaults?api-version=2024-01-01", nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.KeyVault/vaults/vaultone?api-version=2024-01-01", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.KeyVault/vaults/vaultone?api-version=2024-01-01", nil)
	deleteRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusAccepted {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusAccepted)
	}
}

func TestPutKeyVaultRequiresExistingResourceGroup(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/missing/providers/Microsoft.KeyVault/vaults/vaultone?api-version=2024-01-01", strings.NewReader(`{"location":"westus2"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestVirtualNetworkAndSubnetCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertResourceGroup("test-sub", "rg-one", "westus2", "", nil); err != nil {
		t.Fatalf("UpsertResourceGroup() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	createVNetReq := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/virtualNetworks/vnet-one?api-version=2024-01-01", strings.NewReader(`{"location":"westus2","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`))
	createVNetRec := httptest.NewRecorder()
	mux.ServeHTTP(createVNetRec, createVNetReq)
	if createVNetRec.Code != http.StatusAccepted {
		t.Fatalf("create vnet status = %d, want %d", createVNetRec.Code, http.StatusAccepted)
	}

	listVNetReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/virtualNetworks?api-version=2024-01-01", nil)
	listVNetRec := httptest.NewRecorder()
	mux.ServeHTTP(listVNetRec, listVNetReq)
	if listVNetRec.Code != http.StatusOK {
		t.Fatalf("list vnet status = %d, want %d", listVNetRec.Code, http.StatusOK)
	}
	if !strings.Contains(listVNetRec.Body.String(), `"name":"vnet-one"`) {
		t.Fatalf("list vnet body = %q, want vnet name", listVNetRec.Body.String())
	}

	createSubnetReq := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/virtualNetworks/vnet-one/subnets/frontend?api-version=2024-01-01", strings.NewReader(`{"properties":{"addressPrefix":"10.0.1.0/24"}}`))
	createSubnetRec := httptest.NewRecorder()
	mux.ServeHTTP(createSubnetRec, createSubnetReq)
	if createSubnetRec.Code != http.StatusAccepted {
		t.Fatalf("create subnet status = %d, want %d", createSubnetRec.Code, http.StatusAccepted)
	}

	getVNetReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/virtualNetworks/vnet-one?api-version=2024-01-01", nil)
	getVNetRec := httptest.NewRecorder()
	mux.ServeHTTP(getVNetRec, getVNetReq)
	if getVNetRec.Code != http.StatusOK {
		t.Fatalf("get vnet status = %d, want %d", getVNetRec.Code, http.StatusOK)
	}
	if !strings.Contains(getVNetRec.Body.String(), `"subnets":[`) {
		t.Fatalf("get vnet body = %q, want subnets", getVNetRec.Body.String())
	}

	getSubnetReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/virtualNetworks/vnet-one/subnets/frontend?api-version=2024-01-01", nil)
	getSubnetRec := httptest.NewRecorder()
	mux.ServeHTTP(getSubnetRec, getSubnetReq)
	if getSubnetRec.Code != http.StatusOK {
		t.Fatalf("get subnet status = %d, want %d", getSubnetRec.Code, http.StatusOK)
	}

	deleteSubnetReq := httptest.NewRequest(http.MethodDelete, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/virtualNetworks/vnet-one/subnets/frontend?api-version=2024-01-01", nil)
	deleteSubnetRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteSubnetRec, deleteSubnetReq)
	if deleteSubnetRec.Code != http.StatusAccepted {
		t.Fatalf("delete subnet status = %d, want %d", deleteSubnetRec.Code, http.StatusAccepted)
	}

	deleteVNetReq := httptest.NewRequest(http.MethodDelete, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/virtualNetworks/vnet-one?api-version=2024-01-01", nil)
	deleteVNetRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteVNetRec, deleteVNetReq)
	if deleteVNetRec.Code != http.StatusAccepted {
		t.Fatalf("delete vnet status = %d, want %d", deleteVNetRec.Code, http.StatusAccepted)
	}
}

func TestPutSubnetRequiresExistingVirtualNetwork(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertResourceGroup("test-sub", "rg-one", "westus2", "", nil); err != nil {
		t.Fatalf("UpsertResourceGroup() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/virtualNetworks/vnet-one/subnets/frontend?api-version=2024-01-01", strings.NewReader(`{"properties":{"addressPrefix":"10.0.1.0/24"}}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPrivateDNSZoneAndRecordSetCRUD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertResourceGroup("test-sub", "rg-one", "westus2", "", nil); err != nil {
		t.Fatalf("UpsertResourceGroup() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	createZoneReq := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/privateDnsZones/internal.test?api-version=2024-01-01", strings.NewReader(`{"tags":{"env":"test"}}`))
	createZoneRec := httptest.NewRecorder()
	mux.ServeHTTP(createZoneRec, createZoneReq)
	if createZoneRec.Code != http.StatusAccepted {
		t.Fatalf("create zone status = %d, want %d", createZoneRec.Code, http.StatusAccepted)
	}

	listZoneReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/privateDnsZones?api-version=2024-01-01", nil)
	listZoneRec := httptest.NewRecorder()
	mux.ServeHTTP(listZoneRec, listZoneReq)
	if listZoneRec.Code != http.StatusOK {
		t.Fatalf("list zone status = %d, want %d", listZoneRec.Code, http.StatusOK)
	}

	createRecordReq := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/privateDnsZones/internal.test/A/api?api-version=2024-01-01", strings.NewReader(`{"properties":{"TTL":60,"aRecords":[{"ipv4Address":"10.0.0.4"}]}}`))
	createRecordRec := httptest.NewRecorder()
	mux.ServeHTTP(createRecordRec, createRecordReq)
	if createRecordRec.Code != http.StatusAccepted {
		t.Fatalf("create record status = %d, want %d", createRecordRec.Code, http.StatusAccepted)
	}

	var createdRecord map[string]any
	if err := json.Unmarshal(createRecordRec.Body.Bytes(), &createdRecord); err != nil {
		t.Fatalf("json.Unmarshal() create record error = %v", err)
	}
	properties, _ := createdRecord["properties"].(map[string]any)
	if properties["fqdn"] != "api.internal.test." {
		t.Fatalf("fqdn = %v, want %q", properties["fqdn"], "api.internal.test.")
	}

	getRecordReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/privateDnsZones/internal.test/A/api?api-version=2024-01-01", nil)
	getRecordRec := httptest.NewRecorder()
	mux.ServeHTTP(getRecordRec, getRecordReq)
	if getRecordRec.Code != http.StatusOK {
		t.Fatalf("get record status = %d, want %d", getRecordRec.Code, http.StatusOK)
	}

	deleteRecordReq := httptest.NewRequest(http.MethodDelete, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/privateDnsZones/internal.test/A/api?api-version=2024-01-01", nil)
	deleteRecordRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRecordRec, deleteRecordReq)
	if deleteRecordRec.Code != http.StatusAccepted {
		t.Fatalf("delete record status = %d, want %d", deleteRecordRec.Code, http.StatusAccepted)
	}

	deleteZoneReq := httptest.NewRequest(http.MethodDelete, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/privateDnsZones/internal.test?api-version=2024-01-01", nil)
	deleteZoneRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteZoneRec, deleteZoneReq)
	if deleteZoneRec.Code != http.StatusAccepted {
		t.Fatalf("delete zone status = %d, want %d", deleteZoneRec.Code, http.StatusAccepted)
	}
}

func TestPutPrivateDNSRecordRequiresExistingZone(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertResourceGroup("test-sub", "rg-one", "westus2", "", nil); err != nil {
		t.Fatalf("UpsertResourceGroup() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Network/privateDnsZones/internal.test/A/api?api-version=2024-01-01", strings.NewReader(`{"properties":{"TTL":60,"aRecords":[{"ipv4Address":"10.0.0.4"}]}}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestListStorageAccountsRequiresExistingResourceGroup(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/missing/providers/Microsoft.Storage/storageAccounts?api-version=2024-01-01", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDeploymentRoutesPersistFailedRecordAndOperation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertResourceGroup("test-sub", "rg-one", "westus2", "", nil); err != nil {
		t.Fatalf("UpsertResourceGroup() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	createReq := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Resources/deployments/deploy-one?api-version=2024-01-01", strings.NewReader(`{"location":"westus2","properties":{"mode":"Incremental","template":{"resources":[]},"parameters":{"name":{"value":"tiny"}}}}`))
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", createRec.Code, http.StatusAccepted)
	}
	if createRec.Header().Get("Azure-AsyncOperation") == "" {
		t.Fatal("Azure-AsyncOperation header is empty")
	}

	var createBody map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	properties, _ := createBody["properties"].(map[string]any)
	if properties["provisioningState"] != "Failed" {
		t.Fatalf("provisioningState = %v, want %q", properties["provisioningState"], "Failed")
	}
	errorBody, _ := properties["error"].(map[string]any)
	if errorBody["code"] != "DeploymentNotSupported" {
		t.Fatalf("error.code = %v, want %q", errorBody["code"], "DeploymentNotSupported")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Resources/deployments/deploy-one?api-version=2024-01-01", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Resources/deployments?api-version=2024-01-01", nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}
	var listBody struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("json.Unmarshal() list error = %v", err)
	}
	if len(listBody.Value) != 1 {
		t.Fatalf("len(value) = %d, want %d", len(listBody.Value), 1)
	}

	operationPath := createRec.Header().Get("Azure-AsyncOperation")
	opReq := httptest.NewRequest(http.MethodGet, operationPath+"?api-version=2024-01-01", nil)
	opRec := httptest.NewRecorder()
	mux.ServeHTTP(opRec, opReq)
	if opRec.Code != http.StatusOK {
		t.Fatalf("operation status = %d, want %d", opRec.Code, http.StatusOK)
	}
	var opBody map[string]any
	if err := json.Unmarshal(opRec.Body.Bytes(), &opBody); err != nil {
		t.Fatalf("json.Unmarshal() operation error = %v", err)
	}
	if opBody["status"] != "Failed" {
		t.Fatalf("operation status = %v, want %q", opBody["status"], "Failed")
	}
}

func TestDeploymentRoutesCreateSupportedResources(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if _, err := store.UpsertResourceGroup("test-sub", "rg-one", "westus2", "", nil); err != nil {
		t.Fatalf("UpsertResourceGroup() error = %v", err)
	}

	cfg := config.FromEnv()
	mux := http.NewServeMux()
	NewHandler(store, cfg).Register(mux)

	body := `{
		"location":"westus2",
		"properties":{
			"mode":"Incremental",
			"template":{
				"resources":[
					{
						"type":"Microsoft.Storage/storageAccounts",
						"name":"storedeploy",
						"location":"westus2",
						"sku":{"name":"Standard_LRS"}
					},
					{
						"type":"Microsoft.KeyVault/vaults",
						"name":"vaultdeploy",
						"location":"westus2",
						"properties":{"tenantId":"tenant-123"},
						"sku":{"name":"standard"}
					}
				]
			}
		}
	}`
	createReq := httptest.NewRequest(http.MethodPut, "/subscriptions/test-sub/resourceGroups/rg-one/providers/Microsoft.Resources/deployments/deploy-supported?api-version=2024-01-01", strings.NewReader(body))
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", createRec.Code, http.StatusAccepted)
	}

	var createBody map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	properties, _ := createBody["properties"].(map[string]any)
	if properties["provisioningState"] != "Succeeded" {
		t.Fatalf("provisioningState = %v, want %q", properties["provisioningState"], "Succeeded")
	}
	if _, ok := properties["error"]; ok {
		t.Fatalf("error = %v, want absent", properties["error"])
	}
	outputs, _ := properties["outputs"].(map[string]any)
	createdResources, _ := outputs["createdResources"].(map[string]any)
	value, _ := createdResources["value"].([]any)
	if len(value) != 2 {
		t.Fatalf("len(outputs.createdResources.value) = %d, want %d", len(value), 2)
	}

	if _, err := store.GetStorageAccount("test-sub", "rg-one", "storedeploy"); err != nil {
		t.Fatalf("GetStorageAccount() error = %v", err)
	}
	if _, err := store.GetKeyVault("test-sub", "rg-one", "vaultdeploy"); err != nil {
		t.Fatalf("GetKeyVault() error = %v", err)
	}

	operationPath := createRec.Header().Get("Azure-AsyncOperation")
	opReq := httptest.NewRequest(http.MethodGet, operationPath+"?api-version=2024-01-01", nil)
	opRec := httptest.NewRecorder()
	mux.ServeHTTP(opRec, opReq)
	if opRec.Code != http.StatusOK {
		t.Fatalf("operation status = %d, want %d", opRec.Code, http.StatusOK)
	}
	var opBody map[string]any
	if err := json.Unmarshal(opRec.Body.Bytes(), &opBody); err != nil {
		t.Fatalf("json.Unmarshal() operation error = %v", err)
	}
	if opBody["status"] != "Succeeded" {
		t.Fatalf("operation status = %v, want %q", opBody["status"], "Succeeded")
	}
}

func TestGetResourceGroupReturnsNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/test-sub/resourceGroups/missing?api-version=2024-01-01", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var body httpx.CloudErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.Error.Code != "ResourceGroupNotFound" {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, "ResourceGroupNotFound")
	}
}

func TestGetOperationReturnsStatus(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	operation, err := store.CreateOperation("sub-123", "/subscriptions/sub-123/resourceGroups/rg-one", "Microsoft.Resources/resourceGroups/write", "Succeeded")
	if err != nil {
		t.Fatalf("CreateOperation() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/sub-123/providers/Microsoft.Resources/operations/"+operation.ID+"?api-version=2024-01-01", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["status"] != "Succeeded" {
		t.Fatalf("status = %v, want %q", body["status"], "Succeeded")
	}
}
