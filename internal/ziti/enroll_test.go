package ziti

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/openziti/identity"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/sdk-golang/ziti/enroll"
)

func TestEnsureEnrollmentSkipsWhenJWTEmpty(t *testing.T) {
	stubEnrollmentFuncs(t, func(token string) (*ziti.EnrollmentClaims, *jwt.Token, error) {
		t.Fatalf("parse enrollment called unexpectedly")
		return nil, nil, nil
	}, func(flags enroll.EnrollmentFlags) (*ziti.Config, error) {
		t.Fatalf("enroll called unexpectedly")
		return nil, nil
	})

	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "tls.crt")
	keyFile := filepath.Join(tempDir, "tls.key")
	caFile := filepath.Join(tempDir, "ca.crt")

	if err := EnsureEnrollment(certFile, keyFile, caFile, ""); err != nil {
		t.Fatalf("EnsureEnrollment returned error: %v", err)
	}
}

func TestEnsureEnrollmentSkipsWhenCertsExist(t *testing.T) {
	stubEnrollmentFuncs(t, func(token string) (*ziti.EnrollmentClaims, *jwt.Token, error) {
		t.Fatalf("parse enrollment called unexpectedly")
		return nil, nil, nil
	}, func(flags enroll.EnrollmentFlags) (*ziti.Config, error) {
		t.Fatalf("enroll called unexpectedly")
		return nil, nil
	})

	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "tls.crt")
	keyFile := filepath.Join(tempDir, "tls.key")
	caFile := filepath.Join(tempDir, "ca.crt")
	for _, path := range []string{certFile, keyFile, caFile} {
		if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
			t.Fatalf("write test file: %v", err)
		}
	}

	if err := EnsureEnrollment(certFile, keyFile, caFile, filepath.Join(tempDir, "missing.jwt")); err != nil {
		t.Fatalf("EnsureEnrollment returned error: %v", err)
	}

	for _, path := range []string{certFile, keyFile, caFile} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if string(data) != "existing" {
			t.Fatalf("unexpected file contents for %s: %s", path, string(data))
		}
	}
}

func TestEnsureEnrollmentWritesFiles(t *testing.T) {
	stubEnrollmentFuncs(t, func(token string) (*ziti.EnrollmentClaims, *jwt.Token, error) {
		if token != "test-jwt" {
			t.Fatalf("unexpected jwt token: %s", token)
		}
		return &ziti.EnrollmentClaims{}, nil, nil
	}, func(flags enroll.EnrollmentFlags) (*ziti.Config, error) {
		if flags.Token == nil {
			t.Fatalf("expected enrollment claims")
		}
		if !flags.KeyAlg.EC() {
			t.Fatalf("expected EC key algorithm")
		}
		return &ziti.Config{
			ID: identity.Config{
				Cert: "pem:cert-data",
				Key:  "pem:key-data",
				CA:   "pem:ca-data",
			},
		}, nil
	})

	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "tls.crt")
	keyFile := filepath.Join(tempDir, "tls.key")
	caFile := filepath.Join(tempDir, "ca.crt")
	jwtFile := filepath.Join(tempDir, "enrollment.jwt")
	if err := os.WriteFile(jwtFile, []byte("test-jwt\n"), 0o600); err != nil {
		t.Fatalf("write jwt file: %v", err)
	}

	if err := EnsureEnrollment(certFile, keyFile, caFile, jwtFile); err != nil {
		t.Fatalf("EnsureEnrollment returned error: %v", err)
	}

	assertFileContents(t, certFile, "cert-data")
	assertFileContents(t, keyFile, "key-data")
	assertFileContents(t, caFile, "ca-data")

	assertFileMode(t, certFile, 0o600)
	assertFileMode(t, keyFile, 0o600)
	assertFileMode(t, caFile, 0o600)
}

func stubEnrollmentFuncs(
	t *testing.T,
	parse func(string) (*ziti.EnrollmentClaims, *jwt.Token, error),
	enrollFn func(enroll.EnrollmentFlags) (*ziti.Config, error),
) {
	t.Helper()

	originalParse := parseEnrollmentToken
	originalEnroll := enrollIdentity
	parseEnrollmentToken = parse
	enrollIdentity = enrollFn
	t.Cleanup(func() {
		parseEnrollmentToken = originalParse
		enrollIdentity = originalEnroll
	})
}

func assertFileContents(t *testing.T, path, expected string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != expected {
		t.Fatalf("unexpected file contents for %s: %s", path, string(data))
	}
}

func assertFileMode(t *testing.T, path string, expected os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if info.Mode().Perm() != expected {
		t.Fatalf("unexpected file mode for %s: %v", path, info.Mode().Perm())
	}
}
