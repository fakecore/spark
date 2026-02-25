package config

import (
	"testing"

	"spark/internal/conf"
)

func TestValidateBootstrap(t *testing.T) {
	t.Run("ValidBootstrapConfig", func(t *testing.T) {
		bc := &conf.Bootstrap{
			Data: &conf.Data{
				Database: &conf.Data_Database{
					Driver: "postgres",
					Source: "host=127.0.0.1 user=test password=test dbname=test port=5432 sslmode=disable",
				},
				Redis: &conf.Data_Redis{
					Mode:     "auto",
					Addr:     "127.0.0.1:6379",
					Password: "",
					Database: 1,
				},
			},
			Server: &conf.Server{
				Http: &conf.Server_HTTP{
					Addr: "0.0.0.0:9988",
				},
			},
			Auth: &conf.Auth{
				AccessSecret: "test-secret",
			},
		}
		if err := ValidateBootstrap(bc); err != nil {
			t.Fatalf("expected valid config, got %v", err)
		}
	})

	t.Run("ValidConfigWithRedis", func(t *testing.T) {
		bc := &conf.Bootstrap{
			Data: &conf.Data{
				Database: &conf.Data_Database{
					Driver: "postgres",
					Source: "host=127.0.0.1 user=test password=test dbname=test port=5432 sslmode=disable",
				},
				Redis: &conf.Data_Redis{
					Mode:     "external",
					Addr:     "127.0.0.1:6379",
					Password: "",
					Database: 0,
				},
			},
			Server: &conf.Server{
				Http: &conf.Server_HTTP{
					Addr: "0.0.0.0:9988",
				},
			},
		}
		if err := ValidateBootstrap(bc); err != nil {
			t.Fatalf("expected valid config with Redis, got %v", err)
		}
	})
}
