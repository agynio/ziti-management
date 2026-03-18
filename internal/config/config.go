package config

import (
	"fmt"
	"os"
)

type Config struct {
	GRPCAddress       string
	DatabaseURL       string
	ZitiControllerURL string
	ZitiCertFile      string
	ZitiKeyFile       string
	ZitiCAFile        string
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
	return cfg, nil
}
