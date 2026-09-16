package config

import "testing"

func TestLoadFromEnv(t *testing.T) {
	env := map[string]string{
		"API_ADDR":        ":9090",
		"DATABASE_URL":    "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable",
		"MIGRATIONS_PATH": "db/migrations",
		"LOG_LEVEL":       "debug",
	}

	cfg, err := LoadFromEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.APIAddress != ":9090" {
		t.Fatalf("unexpected API address: %q", cfg.APIAddress)
	}

	if cfg.DatabaseURL != env["DATABASE_URL"] {
		t.Fatalf("unexpected database URL: %q", cfg.DatabaseURL)
	}

	if cfg.MigrationsPath != "db/migrations" {
		t.Fatalf("unexpected migrations path: %q", cfg.MigrationsPath)
	}

	if cfg.LogLevel != "debug" {
		t.Fatalf("unexpected log level: %q", cfg.LogLevel)
	}
}

func TestLoadFromEnvUsesDefaults(t *testing.T) {
	cfg, err := LoadFromEnv(func(key string) (string, bool) {
		if key == "DATABASE_URL" {
			return "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.APIAddress != defaultAPIAddress {
		t.Fatalf("unexpected default API address: %q", cfg.APIAddress)
	}

	if cfg.MigrationsPath != defaultMigrationsPath {
		t.Fatalf("unexpected default migrations path: %q", cfg.MigrationsPath)
	}

	if cfg.LogLevel != defaultLogLevel {
		t.Fatalf("unexpected default log level: %q", cfg.LogLevel)
	}
}

func TestLoadFromEnvRequiresDatabaseURL(t *testing.T) {
	_, err := LoadFromEnv(func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("expected missing DATABASE_URL error")
	}
}

func TestLoadFromEnvRejectsEmptyAPIAddress(t *testing.T) {
	_, err := LoadFromEnv(func(key string) (string, bool) {
		switch key {
		case "API_ADDR":
			return " ", true
		case "DATABASE_URL":
			return "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable", true
		default:
			return "", false
		}
	})
	if err == nil {
		t.Fatal("expected empty API_ADDR error")
	}
}

func TestLoadFromEnvRejectsEmptyMigrationsPath(t *testing.T) {
	_, err := LoadFromEnv(func(key string) (string, bool) {
		switch key {
		case "DATABASE_URL":
			return "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable", true
		case "MIGRATIONS_PATH":
			return " ", true
		default:
			return "", false
		}
	})
	if err == nil {
		t.Fatal("expected empty MIGRATIONS_PATH error")
	}
}
