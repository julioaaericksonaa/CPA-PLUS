package managementasset

import (
	"strings"
	"testing"
)

func TestBundledManagementHTMLAvailable(t *testing.T) {
	html := BundledManagementHTML()
	if !strings.Contains(html, "CPA Manager Plus") {
		t.Fatalf("BundledManagementHTML() missing CPA Manager Plus marker")
	}
}

func TestHasBundledManagementHTML(t *testing.T) {
	if !HasBundledManagementHTML() {
		t.Fatalf("HasBundledManagementHTML() = false, want true")
	}
}
