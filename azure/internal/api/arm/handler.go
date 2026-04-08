package arm

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
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
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults", h.listKeyVaults)
	mux.HandleFunc("PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}", h.putKeyVault)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}", h.getKeyVault)
	mux.HandleFunc("DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}", h.deleteKeyVault)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks", h.listVirtualNetworks)
	mux.HandleFunc("PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}", h.putVirtualNetwork)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}", h.getVirtualNetwork)
	mux.HandleFunc("DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}", h.deleteVirtualNetwork)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets", h.listSubnets)
	mux.HandleFunc("PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets/{subnetName}", h.putSubnet)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets/{subnetName}", h.getSubnet)
	mux.HandleFunc("DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets/{subnetName}", h.deleteSubnet)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones", h.listPrivateDNSZones)
	mux.HandleFunc("PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{zoneName}", h.putPrivateDNSZone)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{zoneName}", h.getPrivateDNSZone)
	mux.HandleFunc("DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{zoneName}", h.deletePrivateDNSZone)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{zoneName}/A", h.listPrivateDNSARecordSets)
	mux.HandleFunc("PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{zoneName}/A/{relativeRecordSetName}", h.putPrivateDNSARecordSet)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{zoneName}/A/{relativeRecordSetName}", h.getPrivateDNSARecordSet)
	mux.HandleFunc("DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{zoneName}/A/{relativeRecordSetName}", h.deletePrivateDNSARecordSet)
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

func (h *Handler) listKeyVaults(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	vaults, err := h.store.ListKeyVaults(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"))
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(vaults))
	for _, vault := range vaults {
		value = append(value, h.keyVaultResponse(vault))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putKeyVault(w http.ResponseWriter, r *http.Request) {
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
			TenantID string `json:"tenantId"`
			SKU      struct {
				Name string `json:"name"`
			} `json:"sku"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "request body must be valid JSON")
		return
	}
	if body.Location == "" {
		httpx.WriteCloudError(w, http.StatusBadRequest, "MissingLocation", "key vault location is required")
		return
	}

	vault, err := h.store.UpsertKeyVault(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("vaultName"),
		body.Location,
		body.Properties.TenantID,
		body.Properties.SKU.Name,
		body.Tags,
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		vault.ID,
		"Microsoft.KeyVault/vaults/write",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	httpx.WriteJSON(w, http.StatusAccepted, h.keyVaultResponse(vault))
}

func (h *Handler) getKeyVault(w http.ResponseWriter, r *http.Request) {
	vault, err := h.store.GetKeyVault(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("vaultName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the key vault was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, h.keyVaultResponse(vault))
}

func (h *Handler) deleteKeyVault(w http.ResponseWriter, r *http.Request) {
	vault, err := h.store.GetKeyVault(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("vaultName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the key vault was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	if err := h.store.DeleteKeyVault(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("vaultName")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the key vault was not found")
			return
		}
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		vault.ID,
		"Microsoft.KeyVault/vaults/delete",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) listVirtualNetworks(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	networks, err := h.store.ListVirtualNetworks(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"))
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(networks))
	for _, network := range networks {
		body, err := h.virtualNetworkResponse(network)
		if err != nil {
			httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
			return
		}
		value = append(value, body)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putVirtualNetwork(w http.ResponseWriter, r *http.Request) {
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
			AddressSpace struct {
				AddressPrefixes []string `json:"addressPrefixes"`
			} `json:"addressSpace"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "request body must be valid JSON")
		return
	}
	if body.Location == "" {
		httpx.WriteCloudError(w, http.StatusBadRequest, "MissingLocation", "virtual network location is required")
		return
	}
	if len(body.Properties.AddressSpace.AddressPrefixes) == 0 {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "virtual network requires at least one address prefix")
		return
	}

	network, err := h.store.UpsertVirtualNetwork(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("virtualNetworkName"),
		body.Location,
		body.Properties.AddressSpace.AddressPrefixes,
		body.Tags,
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		network.ID,
		"Microsoft.Network/virtualNetworks/write",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	response, err := h.virtualNetworkResponse(network)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}
	setAsyncHeaders(w, operation)
	httpx.WriteJSON(w, http.StatusAccepted, response)
}

func (h *Handler) getVirtualNetwork(w http.ResponseWriter, r *http.Request) {
	network, err := h.store.GetVirtualNetwork(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("virtualNetworkName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the virtual network was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	response, err := h.virtualNetworkResponse(network)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) deleteVirtualNetwork(w http.ResponseWriter, r *http.Request) {
	network, err := h.store.GetVirtualNetwork(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("virtualNetworkName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the virtual network was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	if err := h.store.DeleteVirtualNetwork(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("virtualNetworkName")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the virtual network was not found")
			return
		}
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		network.ID,
		"Microsoft.Network/virtualNetworks/delete",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) listSubnets(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetVirtualNetwork(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("virtualNetworkName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the virtual network was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	subnets, err := h.store.ListSubnets(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("virtualNetworkName"))
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(subnets))
	for _, subnet := range subnets {
		value = append(value, subnetResponse(subnet))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putSubnet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Properties struct {
			AddressPrefix string `json:"addressPrefix"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "request body must be valid JSON")
		return
	}
	if body.Properties.AddressPrefix == "" {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "subnet requires an addressPrefix")
		return
	}

	subnet, err := h.store.UpsertSubnet(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("virtualNetworkName"),
		r.PathValue("subnetName"),
		body.Properties.AddressPrefix,
	)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the virtual network was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		subnet.ID,
		"Microsoft.Network/virtualNetworks/subnets/write",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	httpx.WriteJSON(w, http.StatusAccepted, subnetResponse(subnet))
}

func (h *Handler) getSubnet(w http.ResponseWriter, r *http.Request) {
	subnet, err := h.store.GetSubnet(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("virtualNetworkName"), r.PathValue("subnetName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the subnet was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, subnetResponse(subnet))
}

func (h *Handler) deleteSubnet(w http.ResponseWriter, r *http.Request) {
	subnet, err := h.store.GetSubnet(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("virtualNetworkName"), r.PathValue("subnetName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the subnet was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	if err := h.store.DeleteSubnet(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("virtualNetworkName"), r.PathValue("subnetName")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the subnet was not found")
			return
		}
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		subnet.ID,
		"Microsoft.Network/virtualNetworks/subnets/delete",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) listPrivateDNSZones(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	zones, err := h.store.ListPrivateDNSZones(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"))
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(zones))
	for _, zone := range zones {
		value = append(value, h.privateDNSZoneResponse(zone))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putPrivateDNSZone(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetResourceGroup(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceGroupNotFound", "the resource group was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	var body struct {
		Tags map[string]string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "request body must be valid JSON")
		return
	}

	zone, err := h.store.UpsertPrivateDNSZone(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("zoneName"),
		body.Tags,
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		zone.ID,
		"Microsoft.Network/privateDnsZones/write",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	httpx.WriteJSON(w, http.StatusAccepted, h.privateDNSZoneResponse(zone))
}

func (h *Handler) getPrivateDNSZone(w http.ResponseWriter, r *http.Request) {
	zone, err := h.store.GetPrivateDNSZone(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("zoneName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the private DNS zone was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, h.privateDNSZoneResponse(zone))
}

func (h *Handler) deletePrivateDNSZone(w http.ResponseWriter, r *http.Request) {
	zone, err := h.store.GetPrivateDNSZone(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("zoneName"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the private DNS zone was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	if err := h.store.DeletePrivateDNSZone(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("zoneName")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the private DNS zone was not found")
			return
		}
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		zone.ID,
		"Microsoft.Network/privateDnsZones/delete",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) listPrivateDNSARecordSets(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.GetPrivateDNSZone(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("zoneName")); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the private DNS zone was not found")
		return
	} else if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	recordSets, err := h.store.ListPrivateDNSARecordSets(r.PathValue("subscriptionId"), r.PathValue("resourceGroupName"), r.PathValue("zoneName"))
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(recordSets))
	for _, recordSet := range recordSets {
		value = append(value, h.privateDNSARecordSetResponse(recordSet))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putPrivateDNSARecordSet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Properties struct {
			TTL      int `json:"TTL"`
			ARecords []struct {
				IPv4Address string `json:"ipv4Address"`
			} `json:"aRecords"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "request body must be valid JSON")
		return
	}

	addresses := make([]string, 0, len(body.Properties.ARecords))
	for _, record := range body.Properties.ARecords {
		if record.IPv4Address != "" {
			if net.ParseIP(record.IPv4Address).To4() == nil {
				httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "private DNS A record set requires valid ipv4Address values")
				return
			}
			addresses = append(addresses, record.IPv4Address)
		}
	}
	if len(addresses) == 0 {
		httpx.WriteCloudError(w, http.StatusBadRequest, "InvalidRequestContent", "private DNS A record set requires at least one ipv4Address")
		return
	}

	recordSet, err := h.store.UpsertPrivateDNSARecordSet(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("zoneName"),
		r.PathValue("relativeRecordSetName"),
		body.Properties.TTL,
		addresses,
	)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the private DNS zone was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		recordSet.ID,
		"Microsoft.Network/privateDnsZones/A/write",
		"Succeeded",
	)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	setAsyncHeaders(w, operation)
	httpx.WriteJSON(w, http.StatusAccepted, h.privateDNSARecordSetResponse(recordSet))
}

func (h *Handler) getPrivateDNSARecordSet(w http.ResponseWriter, r *http.Request) {
	recordSet, err := h.store.GetPrivateDNSARecordSet(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("zoneName"),
		r.PathValue("relativeRecordSetName"),
	)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the private DNS A record set was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, h.privateDNSARecordSetResponse(recordSet))
}

func (h *Handler) deletePrivateDNSARecordSet(w http.ResponseWriter, r *http.Request) {
	recordSet, err := h.store.GetPrivateDNSARecordSet(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("zoneName"),
		r.PathValue("relativeRecordSetName"),
	)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the private DNS A record set was not found")
		return
	}
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	if err := h.store.DeletePrivateDNSARecordSet(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("zoneName"),
		r.PathValue("relativeRecordSetName"),
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.WriteCloudError(w, http.StatusNotFound, "ResourceNotFound", "the private DNS A record set was not found")
			return
		}
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	operation, err := h.store.CreateOperation(
		r.PathValue("subscriptionId"),
		recordSet.ID,
		"Microsoft.Network/privateDnsZones/A/delete",
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

	outputsJSON := `{}`
	provisioningState := "Succeeded"
	errorCode := ""
	errorMessage := ""
	if len(body.Properties.Parameters) > 0 && string(body.Properties.Parameters) != "null" && string(body.Properties.Parameters) != "{}" {
		provisioningState = "Failed"
		errorCode = "DeploymentNotSupported"
		errorMessage = "deployment parameters are not supported"
	} else if len(body.Properties.Template) > 0 && string(body.Properties.Template) != "null" {
		result, err := executeDeploymentTemplate(
			h.store,
			h.cfg,
			r.PathValue("subscriptionId"),
			r.PathValue("resourceGroupName"),
			body.Properties.Template,
		)
		if err != nil {
			provisioningState = "Failed"
			errorCode = "DeploymentNotSupported"
			errorMessage = err.Error()
		} else {
			outputsJSON = result.outputsJSON
		}
	}

	deployment, err := h.store.UpsertDeployment(
		r.PathValue("subscriptionId"),
		r.PathValue("resourceGroupName"),
		r.PathValue("deploymentName"),
		body.Location,
		body.Properties.Mode,
		string(body.Properties.Template),
		string(body.Properties.Parameters),
		outputsJSON,
		provisioningState,
		errorCode,
		errorMessage,
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
		deployment.ProvisioningState,
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

func (h *Handler) keyVaultResponse(vault state.KeyVault) map[string]any {
	return map[string]any{
		"id":       vault.ID,
		"name":     vault.Name,
		"type":     "Microsoft.KeyVault/vaults",
		"location": vault.Location,
		"tags":     vault.Tags,
		"properties": map[string]any{
			"tenantId":          vault.TenantID,
			"provisioningState": vault.ProvisioningState,
			"vaultUri":          fmt.Sprintf("%s/%s", h.cfg.KeyVaultURL(), vault.Name),
		},
		"sku": map[string]string{
			"name": vault.SKUName,
		},
	}
}

func (h *Handler) virtualNetworkResponse(network state.VirtualNetwork) (map[string]any, error) {
	subnets, err := h.store.ListSubnets(network.SubscriptionID, network.ResourceGroupName, network.Name)
	if err != nil {
		return nil, err
	}
	subnetBodies := make([]map[string]any, 0, len(subnets))
	for _, subnet := range subnets {
		subnetBodies = append(subnetBodies, subnetResponse(subnet))
	}
	return map[string]any{
		"id":       network.ID,
		"name":     network.Name,
		"type":     "Microsoft.Network/virtualNetworks",
		"location": network.Location,
		"tags":     network.Tags,
		"properties": map[string]any{
			"addressSpace": map[string]any{
				"addressPrefixes": network.AddressPrefixes,
			},
			"subnets":           subnetBodies,
			"provisioningState": network.ProvisioningState,
		},
	}, nil
}

func subnetResponse(subnet state.Subnet) map[string]any {
	return map[string]any{
		"id":   subnet.ID,
		"name": subnet.Name,
		"type": "Microsoft.Network/virtualNetworks/subnets",
		"properties": map[string]any{
			"addressPrefix":     subnet.AddressPrefix,
			"provisioningState": subnet.ProvisioningState,
		},
	}
}

func (h *Handler) privateDNSZoneResponse(zone state.PrivateDNSZone) map[string]any {
	return map[string]any{
		"id":       zone.ID,
		"name":     zone.Name,
		"type":     "Microsoft.Network/privateDnsZones",
		"location": "global",
		"tags":     zone.Tags,
		"properties": map[string]any{
			"provisioningState": zone.ProvisioningState,
			"fqdn":              zone.Name + ".",
		},
	}
}

func (h *Handler) privateDNSARecordSetResponse(recordSet state.PrivateDNSARecordSet) map[string]any {
	aRecords := make([]map[string]string, 0, len(recordSet.IPv4Addresses))
	for _, address := range recordSet.IPv4Addresses {
		aRecords = append(aRecords, map[string]string{"ipv4Address": address})
	}
	return map[string]any{
		"id":   recordSet.ID,
		"name": recordSet.RelativeName,
		"type": "Microsoft.Network/privateDnsZones/A",
		"properties": map[string]any{
			"fqdn":              privateDNSFQDN(recordSet.ZoneName, recordSet.RelativeName) + ".",
			"TTL":               recordSet.TTL,
			"aRecords":          aRecords,
			"provisioningState": recordSet.ProvisioningState,
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

func privateDNSFQDN(zoneName, relativeName string) string {
	if relativeName == "" || relativeName == "@" {
		return zoneName
	}
	return relativeName + "." + zoneName
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
