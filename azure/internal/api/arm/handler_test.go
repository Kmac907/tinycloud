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
	if len(body.Value) != 3 {
		t.Fatalf("len(value) = %d, want %d", len(body.Value), 3)
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
