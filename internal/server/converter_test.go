package server

import (
	"reflect"
	"strings"
	"testing"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"

	"github.com/agynio/ziti-management/internal/store"
	"github.com/agynio/ziti-management/internal/ziti"
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
			name:  "tracing",
			input: zitimanagementv1.ServiceType_SERVICE_TYPE_TRACING,
			want:  store.ServiceTypeTracing,
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

func TestFromProtoHostV1Config(t *testing.T) {
	tests := []struct {
		name    string
		input   *zitimanagementv1.HostV1Config
		want    *ziti.HostV1ConfigData
		wantErr string
	}{
		{
			name:  "nil",
			input: nil,
			want:  nil,
		},
		{
			name: "valid",
			input: &zitimanagementv1.HostV1Config{
				Protocol: " tcp ",
				Address:  " 10.0.0.1 ",
				Port:     443,
			},
			want: &ziti.HostV1ConfigData{
				Protocol: "tcp",
				Address:  "10.0.0.1",
				Port:     443,
			},
		},
		{
			name: "missing protocol",
			input: &zitimanagementv1.HostV1Config{
				Protocol: " ",
				Address:  "10.0.0.1",
				Port:     443,
			},
			wantErr: "protocol is required",
		},
		{
			name: "missing address",
			input: &zitimanagementv1.HostV1Config{
				Protocol: "tcp",
				Address:  " ",
				Port:     443,
			},
			wantErr: "address is required",
		},
		{
			name: "port too low",
			input: &zitimanagementv1.HostV1Config{
				Protocol: "tcp",
				Address:  "10.0.0.1",
				Port:     0,
			},
			wantErr: "port must be between 1 and 65535",
		},
		{
			name: "port too high",
			input: &zitimanagementv1.HostV1Config{
				Protocol: "tcp",
				Address:  "10.0.0.1",
				Port:     70000,
			},
			wantErr: "port must be between 1 and 65535",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := fromProtoHostV1Config(tc.input)
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
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("expected %#v, got %#v", tc.want, got)
			}
		})
	}
}

func TestFromProtoInterceptV1Config(t *testing.T) {
	tests := []struct {
		name    string
		input   *zitimanagementv1.InterceptV1Config
		want    *ziti.InterceptV1ConfigData
		wantErr string
	}{
		{
			name:  "nil",
			input: nil,
			want:  nil,
		},
		{
			name: "valid",
			input: &zitimanagementv1.InterceptV1Config{
				Protocols: []string{" tcp "},
				Addresses: []string{" example.com "},
				PortRanges: []*zitimanagementv1.PortRange{{
					Low:  80,
					High: 80,
				}},
			},
			want: &ziti.InterceptV1ConfigData{
				Protocols: []string{"tcp"},
				Addresses: []string{"example.com"},
				PortRanges: []ziti.PortRangeData{{
					Low:  80,
					High: 80,
				}},
			},
		},
		{
			name: "missing protocols",
			input: &zitimanagementv1.InterceptV1Config{
				Protocols: nil,
				Addresses: []string{"example.com"},
				PortRanges: []*zitimanagementv1.PortRange{{
					Low:  80,
					High: 80,
				}},
			},
			wantErr: "protocols is required",
		},
		{
			name: "empty protocol",
			input: &zitimanagementv1.InterceptV1Config{
				Protocols: []string{"", "tcp"},
				Addresses: []string{"example.com"},
				PortRanges: []*zitimanagementv1.PortRange{{
					Low:  80,
					High: 80,
				}},
			},
			wantErr: "protocols[0] is empty",
		},
		{
			name: "missing addresses",
			input: &zitimanagementv1.InterceptV1Config{
				Protocols: []string{"tcp"},
				Addresses: nil,
				PortRanges: []*zitimanagementv1.PortRange{{
					Low:  80,
					High: 80,
				}},
			},
			wantErr: "addresses is required",
		},
		{
			name: "empty address",
			input: &zitimanagementv1.InterceptV1Config{
				Protocols: []string{"tcp"},
				Addresses: []string{"", "example.com"},
				PortRanges: []*zitimanagementv1.PortRange{{
					Low:  80,
					High: 80,
				}},
			},
			wantErr: "addresses[0] is empty",
		},
		{
			name: "missing port ranges",
			input: &zitimanagementv1.InterceptV1Config{
				Protocols:  []string{"tcp"},
				Addresses:  []string{"example.com"},
				PortRanges: nil,
			},
			wantErr: "port_ranges is required",
		},
		{
			name: "port range too low",
			input: &zitimanagementv1.InterceptV1Config{
				Protocols: []string{"tcp"},
				Addresses: []string{"example.com"},
				PortRanges: []*zitimanagementv1.PortRange{{
					Low:  0,
					High: 80,
				}},
			},
			wantErr: "port_ranges[0] must be between 1 and 65535",
		},
		{
			name: "port range too high",
			input: &zitimanagementv1.InterceptV1Config{
				Protocols: []string{"tcp"},
				Addresses: []string{"example.com"},
				PortRanges: []*zitimanagementv1.PortRange{{
					Low:  80,
					High: 70000,
				}},
			},
			wantErr: "port_ranges[0] must be between 1 and 65535",
		},
		{
			name: "port range high less than low",
			input: &zitimanagementv1.InterceptV1Config{
				Protocols: []string{"tcp"},
				Addresses: []string{"example.com"},
				PortRanges: []*zitimanagementv1.PortRange{{
					Low:  90,
					High: 80,
				}},
			},
			wantErr: "port_ranges[0] high must be >= low",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := fromProtoInterceptV1Config(tc.input)
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
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("expected %#v, got %#v", tc.want, got)
			}
		})
	}
}
