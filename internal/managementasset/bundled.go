package managementasset

import (
	_ "embed"
	"strings"
)

//go:embed bundled/management.html
var bundledManagementHTML string

func BundledManagementHTML() string {
	return bundledManagementHTML
}

func HasBundledManagementHTML() bool {
	return strings.TrimSpace(bundledManagementHTML) != ""
}
