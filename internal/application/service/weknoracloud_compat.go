package service

import (
	"strings"

	"github.com/Tencent/WeKnora/internal/models/provider"
)

// IsWeKnoraCloudDocReaderAddr keeps legacy parser configuration detection
// working for tenants that already have this engine persisted. New setup and
// credential management are intentionally removed from the product surface.
func IsWeKnoraCloudDocReaderAddr(addr string) bool {
	return strings.TrimSuffix(strings.TrimSpace(addr), "/") == strings.TrimRight(provider.WeKnoraCloudBaseURL, "/")+"/api/v1/doc/reader"
}
