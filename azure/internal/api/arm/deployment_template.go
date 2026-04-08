package arm

import (
	"encoding/json"
	"fmt"
	"strings"

	"tinycloud/internal/config"
	"tinycloud/internal/state"
)

type deploymentResource struct {
	Type       string                `json:"type"`
	Name       string                `json:"name"`
	Location   string                `json:"location"`
	Kind       string                `json:"kind"`
	Tags       map[string]string     `json:"tags"`
	SKU        deploymentResourceSKU `json:"sku"`
	Properties map[string]any        `json:"properties"`
}

type deploymentResourceSKU struct {
	Name string `json:"name"`
}

type deploymentTemplate struct {
	Resources []deploymentResource `json:"resources"`
}

type deploymentResult struct {
	outputsJSON string
}

func executeDeploymentTemplate(store *state.Store, cfg config.Config, subscriptionID, resourceGroupName string, templateJSON []byte) (deploymentResult, error) {
	if len(templateJSON) == 0 || string(templateJSON) == "null" {
		return deploymentResult{outputsJSON: `{}`}, nil
	}

	var template deploymentTemplate
	if err := json.Unmarshal(templateJSON, &template); err != nil {
		return deploymentResult{}, fmt.Errorf("parse template: %w", err)
	}
	if len(template.Resources) == 0 {
		return deploymentResult{}, fmt.Errorf("template must include at least one supported resource")
	}

	created := make([]string, 0, len(template.Resources))
	for _, resource := range template.Resources {
		if isExpression(resource.Name) || isExpression(resource.Location) || isExpression(resource.Kind) || isExpression(resource.SKU.Name) {
			return deploymentResult{}, fmt.Errorf("template expressions are not supported")
		}
		for key, value := range resource.Tags {
			if isExpression(key) || isExpression(value) {
				return deploymentResult{}, fmt.Errorf("template expressions are not supported")
			}
		}

		switch resource.Type {
		case "Microsoft.Storage/storageAccounts":
			account, err := store.UpsertStorageAccount(
				subscriptionID,
				resourceGroupName,
				resource.Name,
				resource.Location,
				resource.Kind,
				resource.SKU.Name,
				resource.Tags,
			)
			if err != nil {
				return deploymentResult{}, err
			}
			created = append(created, account.ID)
		case "Microsoft.KeyVault/vaults":
			tenantID, err := stringProperty(resource.Properties, "tenantId", cfg.TenantID)
			if err != nil {
				return deploymentResult{}, err
			}
			account, err := store.UpsertKeyVault(
				subscriptionID,
				resourceGroupName,
				resource.Name,
				resource.Location,
				tenantID,
				resource.SKU.Name,
				resource.Tags,
			)
			if err != nil {
				return deploymentResult{}, err
			}
			created = append(created, account.ID)
		default:
			return deploymentResult{}, fmt.Errorf("resource type %q is not supported", resource.Type)
		}
	}

	outputs := map[string]any{
		"createdResources": map[string]any{
			"type":  "Array",
			"value": created,
		},
	}
	body, err := json.Marshal(outputs)
	if err != nil {
		return deploymentResult{}, fmt.Errorf("marshal outputs: %w", err)
	}

	return deploymentResult{outputsJSON: string(body)}, nil
}

func stringProperty(properties map[string]any, key, fallback string) (string, error) {
	if properties == nil {
		return fallback, nil
	}
	value, ok := properties[key]
	if !ok || value == nil {
		return fallback, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("property %q must be a string", key)
	}
	if isExpression(text) {
		return "", fmt.Errorf("template expressions are not supported")
	}
	if text == "" {
		return fallback, nil
	}
	return text, nil
}

func isExpression(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]")
}
