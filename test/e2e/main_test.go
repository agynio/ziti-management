//go:build e2e

package e2e

import (
	"os"
	"strings"
	"testing"
)

var zitiManagementAddress = envOrDefault("ZITI_MANAGEMENT_ADDRESS", "ziti-management:50051")

func envOrDefault(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
