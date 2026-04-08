package arm

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"tinycloud/internal/config"
	"tinycloud/internal/httpx"
	"tinycloud/internal/state"
)

type Handler struct {
	store *state.Store
	cfg   config.Config
}

func NewHandler(store *state.Store, cfg config.Config) *Handler {
	return &Handler{store: store, cfg: cfg}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /tenants", h.listTenants)
	mux.HandleFunc("GET /subscriptions", h.listSubscriptions)
	mux.HandleFunc("GET /providers", h.listProviders)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/providers", h.listProviders)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/providers/{namespace}", h.getProvider)
	mux.HandleFunc("POST /subscriptions/{subscriptionId}/providers/{namespace}/register", h.registerProvider)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/providers/Microsoft.Resources/operations/{operationId}", h.getOperation)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups", h.listResourceGroups)
	mux.HandleFunc("PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}", h.putResourceGroup)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}", h.getResourceGroup)
	mux.HandleFunc("DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}", h.deleteResourceGroup)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts", h.listStorageAccounts)
	mux.HandleFunc("PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{accountName}", h.putStorageAccount)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{accountName}", h.getStorageAccount)
	mux.HandleFunc("DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{accountName}", h.deleteStorageAccount)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Resources/deployments", h.listDeployments)
	mux.HandleFunc("PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Resources/deployments/{deploymentName}", h.putDeployment)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Resources/deployments/{deploymentName}", h.getDeployment)
}

func (h *Handler) listTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.store.ListTenants()
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(tenants))
	for _, tenant := range tenants {
		value = append(value, map[string]any{
			"id":       fmt.Sprintf("/tenants/%s", tenant.ID),
			"tenantId": tenant.ID,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := h.store.ListSubscriptions()
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		value = append(value, map[string]any{
			"id":             fmt.Sprintf("/subscriptions/%s", subscription.ID),
			"subscriptionId": subscription.ID,
			"tenantId":       subscription.TenantID,
			"displayName":    "TinyCloud Local Subscription",
			"state":          "Enabled",
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) listProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.store.ListProviders()
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		value = append(value, map[string]any{
			"namespace":         provider.Namespace,
			"registrationState": provider.RegistrationState,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) getProvider(w http.ResponseWriter, r *http.Request) {
	provider, err := h.store.GetProvider(r.PathValue("namespace"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ProviderNotFound", "the provider was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, providerResponse(provider))
}

func (h *Handler) registerProvider(w http.ResponseWriter, r *http.Request) {
	provider, err := h.store.RegisterProvider(r.PathValue("namespace"))
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, providerResponse(provider))
}

func (h *Handler) listResourceGroups(w http.ResponseWriter, r *http.Request) {
	resourceGroups, err := h.store.ListResourceGroups(r.PathValue("subscriptionId"))
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(resourceGroups))
	for _, resourceGroup := range resourceGroups {
		value = append(value, resourceGroupResponse(resourceGroup))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putResourceGroup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Location  string            `json:"location"`
		Tags      map[string]string `json:"tags"`
		ManagedBy string            `json:"managedBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "request body must be valid JSON")
		return
	}
	if body.Location == "" {
		httpx.WriteCloudError(w, http.StatusBadRequest, "MissingLocation", "resource group location is required")
		return
	}

	resourceGroup, err := h.store.UpsertResourceGroup(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		body.Location,
		body.ManagedBy,
		body.Tags,
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		resourceGroup.ID,
		"Microsoft.Resources/resourceGroups/write",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	httpx.WriteJSON(w, http.StatusAccepted, resourceGroupResponse(resourceGroup))
}

func (h *Handler) getResourceGroup(w http.ResponseWriter, r *http.Request) {
	resourceGroup, err := h.store.GetResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, resourceGroupResponse(resourceGroup))
}

func (h *Handler) deleteResourceGroup(w http.ResponseWriter, r *http.Request) {
	err := h.store.DeleteResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	resourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"))
	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		resourceID,
		"Microsoft.Resources/resourceGroups/delete",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) getOperation(w http.ResponseWriter, r *http.Request) {
	operation, err := h.store.GetOperation(r.PathValue("subscriptionId"), r.PathValue("operationId"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "OperationNotFound", "the operation was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	body := map[string]any{
		"id":         operation.ID,
		"name":       operation.ID,
		"status":     operation.Status,
		"startTime":  operation.CreatedAt,
		"endTime":    operation.UpdatedAt,
		"properties": map[string]any{"resourceId": operation.ResourceID, "operation": operation.Operation},
	}
	if operation.ErrorCode != "" || operation.ErrorMessage != "" {
		body["error"] = map[string]string{
			"code":    operation.ErrorCode,
			"message": operation.ErrorMessage,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, body)
}

func (h *Handler) listStorageAccounts(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	accounts, err := h.store.ListStorageAccounts(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"))
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(accounts))
	for _, account := range accounts {
		value = append(value, h.storageAccountResponse(account))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putStorageAccount(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	var body struct {
		Location string            `json:"location"`
		Kind     string            `json:"kind"`
		Tags     map[string]string `json:"tags"`
		SKU      struct {
			Name string `json:"name"`
		} `json:"sku"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "request body must be valid JSON")
		return
	}
	if body.Location == "" {
		httpx.WriteCloudError(w, http.StatusBadRequest, "MissingLocation", "storage account location is required")
		return
	}

	account, err := h.store.UpsertStorageAccount(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("accountName"),
		body.Location,
		body.Kind,
		body.SKU.Name,
		body.Tags,
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		account.ID,
		"Microsoft.Storage/storageAccounts/write",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	httpx.WriteJSON(w, http.StatusAccepted, h.storageAccountResponse(account))
}

func (h *Handler) getStorageAccount(w http.ResponseWriter, r *http.Request) {
	account, err := h.store.GetStorageAccount(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("accountName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the storage account was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, h.storageAccountResponse(account))
}

func (h *Handler) deleteStorageAccount(w http.ResponseWriter, r *http.Request) {
	account, err := h.store.GetStorageAccount(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("accountName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the storage account was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	if err := h.store.DeleteStorageAccount(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("accountName")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the storage account was not found")
			return
		}
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		account.ID,
		"Microsoft.Storage/storageAccounts/delete",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	deployments, err := h.store.ListDeployments(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"))
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(deployments))
	for _, deployment := range deployments {
		value = append(value, deploymentResponse(deployment))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putDeployment(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	var body struct {
		Location   string            `json:"location"`
		Tags       map[string]string `json:"tags"`
		Properties struct {
			Mode       string          `json:"mode"`
			Template   json.RawMessage `json:"template"`
			Parameters json.RawMessage `json:"parameters"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "request body must be valid JSON")
		return
	}

	deployment, err := h.store.UpsertDeployment(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("deploymentName"),
		body.Location,
		body.Properties.Mode,
		string(body.Properties.Template),
		string(body.Properties.Parameters),
		`{}`,
		"Failed",
		"DeploymentNotSupported",
		"ARM deployment execution is not implemented yet",
		body.Tags,
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperationResult(
		r.PathValue("subscriptionId"),
		deployment.ID,
		"Microsoft.Resources/deployments/write",
		"Failed",
		deployment.ErrorCode,
		deployment.ErrorMessage,
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	httpx.WriteJSON(w, http.StatusAccepted, deploymentResponse(deployment))
}

func (h *Handler) getDeployment(w http.ResponseWriter, r *http.Request) {
	deployment, err := h.store.GetDeployment(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("deploymentName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "DeploymentNotFound", "the deployment was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, deploymentResponse(deployment))
}

func resourceGroupResponse(resourceGroup state.ResourceGroup) map[string]any {
	return map[string]any{
		"id":        resourceGroup.ID,
		"name":      resourceGroup.Name,
		"type":      resourceGroup.Type,
		"location":  resourceGroup.Location,
		"tags":      resourceGroup.Tags,
		"managedBy": resourceGroup.ManagedBy,
		"properties": map[string]any{
			"provisioningState": resourceGroup.ProvisioningState,
		},
	}
}

func providerResponse(provider state.Provider) map[string]any {
	return map[string]any{
		"namespace":         provider.Namespace,
		"registrationState": provider.RegistrationState,
	}
}

func (h *Handler) storageAccountResponse(account state.StorageAccount) map[string]any {
	return map[string]any{
		"id":       account.ID,
		"name":     account.Name,
		"type":     "Microsoft.Storage/storageAccounts",
		"location": account.Location,
		"kind":     account.Kind,
		"sku": map[string]string{
			"name": account.SKUName,
		},
		"tags": account.Tags,
		"properties": map[string]any{
			"provisioningState": account.ProvisioningState,
			"primaryEndpoints": map[string]string{
				"blob": fmt.Sprintf("%s/%s", h.cfg.BlobURL(), account.Name),
			},
		},
	}
}

func deploymentResponse(deployment state.Deployment) map[string]any {
	properties := map[string]any{
		"mode":              deployment.Mode,
		"provisioningState": deployment.ProvisioningState,
	}
	if deployment.OutputsJSON != "" && deployment.OutputsJSON != "{}" {
		var outputs any
		if err := json.Unmarshal([]byte(deployment.OutputsJSON), &outputs); err == nil {
			properties["outputs"] = outputs
		}
	}
	if deployment.ErrorCode != "" || deployment.ErrorMessage != "" {
		properties["error"] = map[string]string{
			"code":    deployment.ErrorCode,
			"message": deployment.ErrorMessage,
		}
	}

	return map[string]any{
		"id":         deployment.ID,
		"name":       deployment.Name,
		"type":       "Microsoft.Resources/deployments",
		"location":   deployment.Location,
		"tags":       deployment.Tags,
		"properties": properties,
	}
}

func setAsyncHeaders(w http.ResponseWriter, operation state.Operation) {
	pollURL := fmt.Sprintf(
		"/subscriptions/%s/providers/Microsoft.Resources/operations/%s",
		operation.SubscriptionID,
		operation.ID,
	)
	w.Header().Set("Azure-AsyncOperation", pollURL)
	w.Header().Set("Location", pollURL)
	w.Header().Set("Retry-After", "1")
}
