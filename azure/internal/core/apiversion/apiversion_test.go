package apiversion

import "testing"

func TestParseAcceptsStableVersion(t *testing.T) {
	t.Parallel()

	got, err := Parse("2024-01-01")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got != "2024-01-01" {
		t.Fatalf("Parse() = %q, want %q", got, "2024-01-01")
	}
}

func TestParseAcceptsPreviewVersion(t *testing.T) {
	t.Parallel()

	got, err := Parse("2024-01-01-preview")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got != "2024-01-01-preview" {
		t.Fatalf("Parse() = %q, want %q", got, "2024-01-01-preview")
	}
}

func TestParseRejectsMissingVersion(t *testing.T) {
	t.Parallel()

	if _, err := Parse(" "); err != ErrMissing {
		t.Fatalf("Parse() error = %v, want %v", err, ErrMissing)
	}
}

func TestParseRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	if _, err := Parse("v1"); err != ErrInvalid {
		t.Fatalf("Parse() error = %v, want %v", err, ErrInvalid)
	}
}
