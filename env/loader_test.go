package env

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
)

type TestDatabaseConfig struct {
	Host     string `mapstructure:"host" required:"true"`
	Port     int    `mapstructure:"port" required:"true"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type TestAppConfig struct {
	Name     string             `mapstructure:"name" required:"true"`
	Port     int                `mapstructure:"port"`
	Database TestDatabaseConfig `mapstructure:"database"`
}

func TestLoad_RequiredValidationFailure(t *testing.T) {
	// Empty config, Name is required
	var cfg TestAppConfig
	err := Load("TEST_APP", &cfg)
	if err == nil {
		t.Fatal("expected error due to missing required fields, got nil")
	}

	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) {
		t.Fatalf("expected stderr.Error, got %T", err)
	}

	if stdErr.Code != stderr.CodeValidationFailed {
		t.Errorf("expected code CodeValidationFailed, got %v", stdErr.Code)
	}

	if len(stdErr.Details) == 0 {
		t.Errorf("expected validation details for missing fields")
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	os.Setenv("MYAPP_NAME", "GocoreService")
	os.Setenv("MYAPP_PORT", "8080")
	os.Setenv("MYAPP_DATABASE_HOST", "localhost")
	os.Setenv("MYAPP_DATABASE_PORT", "5432")
	defer func() {
		os.Unsetenv("MYAPP_NAME")
		os.Unsetenv("MYAPP_PORT")
		os.Unsetenv("MYAPP_DATABASE_HOST")
		os.Unsetenv("MYAPP_DATABASE_PORT")
	}()

	var cfg TestAppConfig
	err := Load("MYAPP", &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "GocoreService" {
		t.Errorf("expected name GocoreService, got %s", cfg.Name)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("expected db host localhost, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("expected db port 5432, got %d", cfg.Database.Port)
	}
}

func TestLoad_FileWithEnvOverride(t *testing.T) {
	tempDir := t.TempDir()
	yamlContent := `
name: "FileApp"
port: 3000
database:
  host: "file-db"
  port: 5432
`
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Environment variable should override file setting
	os.Setenv("OVERRIDE_APP_PORT", "9090")
	defer os.Unsetenv("OVERRIDE_APP_PORT")

	var cfg TestAppConfig
	err := Load("OVERRIDE_APP", &cfg, WithConfigFile(configFile))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "FileApp" {
		t.Errorf("expected name FileApp, got %s", cfg.Name)
	}
	// Overridden by env
	if cfg.Port != 9090 {
		t.Errorf("expected overridden port 9090, got %d", cfg.Port)
	}
	if cfg.Database.Host != "file-db" {
		t.Errorf("expected host file-db, got %s", cfg.Database.Host)
	}
}

func TestLoad_DotEnvFile(t *testing.T) {
	tempDir := t.TempDir()
	dotEnvContent := `
# Sample dotenv
DOTENV_APP_NAME="DotEnvApp"
DOTENV_APP_PORT=4000
DOTENV_APP_DATABASE_HOST="dotenv-host"
DOTENV_APP_DATABASE_PORT=3306
`
	dotEnvFile := filepath.Join(tempDir, ".env.test")
	if err := os.WriteFile(dotEnvFile, []byte(dotEnvContent), 0600); err != nil {
		t.Fatalf("failed to write dotenv file: %v", err)
	}
	defer func() {
		os.Unsetenv("DOTENV_APP_NAME")
		os.Unsetenv("DOTENV_APP_PORT")
		os.Unsetenv("DOTENV_APP_DATABASE_HOST")
		os.Unsetenv("DOTENV_APP_DATABASE_PORT")
	}()

	var cfg TestAppConfig
	err := Load("DOTENV_APP", &cfg, WithEnvFile(dotEnvFile))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "DotEnvApp" {
		t.Errorf("expected name DotEnvApp, got %s", cfg.Name)
	}
	if cfg.Port != 4000 {
		t.Errorf("expected port 4000, got %d", cfg.Port)
	}
	if cfg.Database.Host != "dotenv-host" {
		t.Errorf("expected host dotenv-host, got %s", cfg.Database.Host)
	}
}

func TestLoad_TargetValidation(t *testing.T) {
	if err := Load("PREFIX", nil); err == nil {
		t.Error("expected error for nil target")
	}

	nonPtr := TestAppConfig{}
	if err := Load("PREFIX", nonPtr); err == nil {
		t.Error("expected error for non-pointer target")
	}
}
