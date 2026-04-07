package resourceid

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalid = errors.New("invalid resource id")

type ID struct {
	SubscriptionID string
	ResourceGroup  string
	Provider       string
	Types          []string
	Names          []string
}

func Parse(path string) (ID, error) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return ID{}, ErrInvalid
	}

	segments := strings.Split(trimmed, "/")
	id := ID{}
	i := 0

	for i < len(segments) {
		switch strings.ToLower(segments[i]) {
		case "subscriptions":
			if i+1 >= len(segments) || segments[i+1] == "" {
				return ID{}, fmt.Errorf("%w: missing subscription id", ErrInvalid)
			}
			id.SubscriptionID = segments[i+1]
			i += 2
		case "resourcegroups":
			if i+1 >= len(segments) || segments[i+1] == "" {
				return ID{}, fmt.Errorf("%w: missing resource group name", ErrInvalid)
			}
			id.ResourceGroup = segments[i+1]
			i += 2
		case "providers":
			if i+1 >= len(segments) || segments[i+1] == "" {
				return ID{}, fmt.Errorf("%w: missing provider namespace", ErrInvalid)
			}
			id.Provider = segments[i+1]
			i += 2
			remaining := len(segments) - i
			if remaining == 0 {
				return id, nil
			}
			if remaining%2 != 0 {
				return ID{}, fmt.Errorf("%w: provider resources must alternate type/name segments", ErrInvalid)
			}
			for i < len(segments) {
				resourceType := segments[i]
				resourceName := segments[i+1]
				if resourceType == "" || resourceName == "" {
					return ID{}, fmt.Errorf("%w: empty type or name segment", ErrInvalid)
				}
				id.Types = append(id.Types, resourceType)
				id.Names = append(id.Names, resourceName)
				i += 2
			}
		default:
			return ID{}, fmt.Errorf("%w: unexpected segment %q", ErrInvalid, segments[i])
		}
	}

	return id, nil
}

func (id ID) IsResourceGroupScope() bool {
	return id.SubscriptionID != "" && id.ResourceGroup != "" && id.Provider == ""
}

func (id ID) IsProviderResource() bool {
	return id.SubscriptionID != "" && id.Provider != "" && len(id.Types) > 0 && len(id.Types) == len(id.Names)
}
