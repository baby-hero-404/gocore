# Package `env`

`env` provides structured configuration loading powered by Viper, with support for environment variables, `.env` files, YAML/JSON configuration files, and fail-fast startup validation for required fields.

## Features

- **Environment-first**: Automatically maps nested structs to environment variables (e.g. `APP_DATABASE_HOST` maps to `cfg.Database.Host`).
- **Fail-Fast Validation**: Validates fields marked with `required:"true"` or `validate:"required"`, returning typed `stderr.Error` (code `1001`) with clear field causes.
- **Viper Quirks Solved**: Recursively inspects and binds struct fields so environment variables populate cleanly even without a config file on disk.
- **Flexible Sources**: Combine `.env` files, config files (`config.yaml`), and environment variable overrides seamlessly.

## Usage

```go
package main

import (
    "log"
    "github.com/baby-hero-404/gocore/env"
)

type Config struct {
    Server struct {
        Port int `mapstructure:"port" required:"true"`
    } `mapstructure:"server"`
    Database struct {
        DSN string `mapstructure:"dsn" required:"true"`
    } `mapstructure:"database"`
}

func main() {
    var cfg Config
    err := env.Load("MYAPP", &cfg, env.WithConfigFile("configs/config.yaml"), env.WithEnvFile(".env"))
    if err != nil {
        log.Fatalf("Config load error: %v", err)
    }
}
```
