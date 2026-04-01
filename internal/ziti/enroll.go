package ziti

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/sdk-golang/ziti/enroll"
)

const pemPrefix = "pem:"

var parseEnrollmentToken = enroll.ParseToken
var enrollIdentity = enroll.Enroll

func EnsureEnrollment(certFile, keyFile, caFile, jwtFile string) error {
	if jwtFile == "" {
		return nil
	}

	certsExist, err := certFilesExist(certFile, keyFile, caFile)
	if err != nil {
		return err
	}
	if certsExist {
		return nil
	}

	jwtBytes, err := os.ReadFile(jwtFile)
	if err != nil {
		return fmt.Errorf("read enrollment jwt: %w", err)
	}
	jwt := strings.TrimSpace(string(jwtBytes))
	if jwt == "" {
		return errors.New("enrollment jwt is empty")
	}

	claims, _, err := parseEnrollmentToken(jwt)
	if err != nil {
		return fmt.Errorf("parse enrollment jwt: %w", err)
	}

	var keyAlg ziti.KeyAlgVar
	if err := keyAlg.Set("EC"); err != nil {
		return fmt.Errorf("set key algorithm: %w", err)
	}

	cfg, err := enrollIdentity(enroll.EnrollmentFlags{
		Token:  claims,
		KeyAlg: keyAlg,
	})
	if err != nil {
		return fmt.Errorf("enroll identity: %w", err)
	}

	certPEM, keyPEM, caPEM, err := extractPEMCredentials(cfg)
	if err != nil {
		return err
	}

	if err := writePEMFile(certFile, certPEM); err != nil {
		return fmt.Errorf("write ziti cert: %w", err)
	}
	if err := writePEMFile(keyFile, keyPEM); err != nil {
		return fmt.Errorf("write ziti key: %w", err)
	}
	if err := writePEMFile(caFile, caPEM); err != nil {
		return fmt.Errorf("write ziti ca: %w", err)
	}

	return nil
}

func certFilesExist(certFile, keyFile, caFile string) (bool, error) {
	for _, entry := range []struct {
		label string
		path  string
	}{
		{label: "cert", path: certFile},
		{label: "key", path: keyFile},
		{label: "ca", path: caFile},
	} {
		if entry.path == "" {
			return false, fmt.Errorf("ziti %s file path is empty", entry.label)
		}
		_, err := os.Stat(entry.path)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, fmt.Errorf("stat ziti %s file: %w", entry.label, err)
		}
	}
	return true, nil
}

func extractPEMCredentials(cfg *ziti.Config) (string, string, string, error) {
	if cfg == nil {
		return "", "", "", errors.New("enrollment returned nil config")
	}

	certPEM, err := stripPEMPrefix(cfg.ID.Cert, "cert")
	if err != nil {
		return "", "", "", err
	}
	keyPEM, err := stripPEMPrefix(cfg.ID.Key, "key")
	if err != nil {
		return "", "", "", err
	}
	caPEM, err := stripPEMPrefix(cfg.ID.CA, "ca")
	if err != nil {
		return "", "", "", err
	}
	return certPEM, keyPEM, caPEM, nil
}

func stripPEMPrefix(value, label string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("enrollment config missing %s", label)
	}
	if !strings.HasPrefix(value, pemPrefix) {
		return "", fmt.Errorf("enrollment config %s missing %s prefix", label, pemPrefix)
	}
	trimmed := strings.TrimPrefix(value, pemPrefix)
	if trimmed == "" {
		return "", fmt.Errorf("enrollment config %s is empty", label)
	}
	return trimmed, nil
}

func writePEMFile(path, contents string) error {
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
