package server

import (
	"reflect"
	"strings"
	"testing"

	"github.com/agynio/ziti-management/internal/store"
)

func TestServiceIdentityConfig(t *testing.T) {
	tests := []struct {
		name       string
		service    store.ServiceType
		wantPrefix string
		wantRoles  []string
		wantErr    string
	}{
		{
			name:       "llm proxy",
			service:    store.ServiceTypeLLMProxy,
			wantPrefix: "svc-llm-proxy-",
			wantRoles:  []string{"llm-proxy-hosts"},
		},
		{
			name:       "tracing",
			service:    store.ServiceTypeTracing,
			wantPrefix: "svc-tracing-",
			wantRoles:  []string{"tracing-hosts"},
		},
		{
			name:    "unspecified",
			service: store.ServiceTypeUnspecified,
			wantErr: "service type unspecified",
		},
		{
			name:    "unknown",
			service: store.ServiceType(99),
			wantErr: "unknown service type",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			name, roles, err := serviceIdentityConfig(tc.service)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("unexpected error: %v", err)
				}
				if name != "" {
					t.Fatalf("expected empty name, got %q", name)
				}
				if roles != nil {
					t.Fatalf("expected nil roles, got %v", roles)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasPrefix(name, tc.wantPrefix) {
				t.Fatalf("expected name prefix %q, got %q", tc.wantPrefix, name)
			}
			suffix := strings.TrimPrefix(name, tc.wantPrefix)
			if len(suffix) != 8 {
				t.Fatalf("expected 8-char suffix, got %q", suffix)
			}
			if !reflect.DeepEqual(roles, tc.wantRoles) {
				t.Fatalf("expected roles %v, got %v", tc.wantRoles, roles)
			}
		})
	}
}
