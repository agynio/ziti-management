package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	GRPCAddress               string
	DatabaseURL               string
	ZitiControllerURL         string
	ZitiCertFile              string
	ZitiKeyFile               string
	ZitiCAFile                string
	ServiceIdentityLeaseTTL   time.Duration
	ServiceIdentityGCInterval time.Duration
}

func FromEnv() (Config, error) {
	cfg := Config{}
	cfg.GRPCAddress = os.Getenv("GRPC_ADDRESS")
	if cfg.GRPCAddress == "" {
		cfg.GRPCAddress = ":50051"
	}
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL must be set")
	}
	cfg.ZitiControllerURL = os.Getenv("ZITI_CONTROLLER_URL")
	if cfg.ZitiControllerURL == "" {
		return Config{}, fmt.Errorf("ZITI_CONTROLLER_URL must be set")
	}
	cfg.ZitiCertFile = os.Getenv("ZITI_CERT_FILE")
	if cfg.ZitiCertFile == "" {
		return Config{}, fmt.Errorf("ZITI_CERT_FILE must be set")
	}
	cfg.ZitiKeyFile = os.Getenv("ZITI_KEY_FILE")
	if cfg.ZitiKeyFile == "" {
		return Config{}, fmt.Errorf("ZITI_KEY_FILE must be set")
	}
	cfg.ZitiCAFile = os.Getenv("ZITI_CA_FILE")
	if cfg.ZitiCAFile == "" {
		return Config{}, fmt.Errorf("ZITI_CA_FILE must be set")
	}
	leaseTTL, err := durationFromEnv("SERVICE_IDENTITY_LEASE_TTL", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	gcInterval, err := durationFromEnv("SERVICE_IDENTITY_GC_INTERVAL", time.Minute)
	if err != nil {
		return Config{}, err
	}
	cfg.ServiceIdentityLeaseTTL = leaseTTL
	cfg.ServiceIdentityGCInterval = gcInterval
	return cfg, nil
}

func durationFromEnv(key string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", key)
	}
	return parsed, nil
}
