package resourceid

import "testing"

func TestParseResourceGroupID(t *testing.T) {
	t.Parallel()

	id, err := Parse("/subscriptions/sub-123/resourceGroups/rg-test")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if id.SubscriptionID != "sub-123" {
		t.Fatalf("SubscriptionID = %q, want %q", id.SubscriptionID, "sub-123")
	}
	if id.ResourceGroup != "rg-test" {
		t.Fatalf("ResourceGroup = %q, want %q", id.ResourceGroup, "rg-test")
	}
	if !id.IsResourceGroupScope() {
		t.Fatal("IsResourceGroupScope() = false, want true")
	}
}

func TestParseProviderResourceID(t *testing.T) {
	t.Parallel()

	id, err := Parse("/subscriptions/sub-123/resourceGroups/rg-test/providers/Microsoft.Storage/storageAccounts/account1/blobServices/default")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if id.Provider != "Microsoft.Storage" {
		t.Fatalf("Provider = %q, want %q", id.Provider, "Microsoft.Storage")
	}
	if len(id.Types) != 2 || id.Types[0] != "storageAccounts" || id.Types[1] != "blobServices" {
		t.Fatalf("Types = %#v, want storageAccounts/blobServices", id.Types)
	}
	if len(id.Names) != 2 || id.Names[0] != "account1" || id.Names[1] != "default" {
		t.Fatalf("Names = %#v, want account1/default", id.Names)
	}
	if !id.IsProviderResource() {
		t.Fatal("IsProviderResource() = false, want true")
	}
}

func TestParseRejectsUnexpectedSegments(t *testing.T) {
	t.Parallel()

	if _, err := Parse("/tenants/test"); err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}

func TestParseRejectsOddProviderSegments(t *testing.T) {
	t.Parallel()

	if _, err := Parse("/subscriptions/sub-123/providers/Microsoft.Storage/storageAccounts"); err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}
