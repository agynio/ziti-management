package server

import (
	"strings"
	"testing"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"

	"github.com/agynio/ziti-management/internal/store"
)

func TestFromProtoServiceType(t *testing.T) {
	tests := []struct {
		name    string
		input   zitimanagementv1.ServiceType
		want    store.ServiceType
		wantErr string
	}{
		{
			name:  "gateway",
			input: zitimanagementv1.ServiceType_SERVICE_TYPE_GATEWAY,
			want:  store.ServiceTypeGateway,
		},
		{
			name:  "orchestrator",
			input: zitimanagementv1.ServiceType_SERVICE_TYPE_ORCHESTRATOR,
			want:  store.ServiceTypeOrchestrator,
		},
		{
			name:  "llm proxy",
			input: zitimanagementv1.ServiceType_SERVICE_TYPE_LLM_PROXY,
			want:  store.ServiceTypeLLMProxy,
		},
		{
			name:    "unspecified",
			input:   zitimanagementv1.ServiceType_SERVICE_TYPE_UNSPECIFIED,
			want:    store.ServiceTypeUnspecified,
			wantErr: "service type unspecified",
		},
		{
			name:    "unknown",
			input:   zitimanagementv1.ServiceType(99),
			want:    store.ServiceTypeUnspecified,
			wantErr: "unknown service type",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := fromProtoServiceType(tc.input)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestFromProtoServicePolicyType(t *testing.T) {
	tests := []struct {
		name    string
		input   zitimanagementv1.ServicePolicyType
		want    string
		wantErr string
	}{
		{
			name:  "bind",
			input: zitimanagementv1.ServicePolicyType_SERVICE_POLICY_TYPE_BIND,
			want:  "Bind",
		},
		{
			name:  "dial",
			input: zitimanagementv1.ServicePolicyType_SERVICE_POLICY_TYPE_DIAL,
			want:  "Dial",
		},
		{
			name:    "unspecified",
			input:   zitimanagementv1.ServicePolicyType_SERVICE_POLICY_TYPE_UNSPECIFIED,
			wantErr: "service policy type unspecified",
		},
		{
			name:    "unknown",
			input:   zitimanagementv1.ServicePolicyType(99),
			wantErr: "unknown service policy type",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := fromProtoServicePolicyType(tc.input)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
