package arm

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"tinycloud/internal/httpx"
	"tinycloud/internal/state"
)

type Handler struct {
	store *state.Store
}

func NewHandler(store *state.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /subscriptions", h.listSubscriptions)
	mux.HandleFunc("GET /providers", h.listProviders)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/providers/Microsoft.Resources/operations/{operationId}", h.getOperation)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups", h.listResourceGroups)
	mux.HandleFunc("PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}", h.putResourceGroup)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}", h.getResourceGroup)
	mux.HandleFunc("DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}", h.deleteResourceGroup)
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
