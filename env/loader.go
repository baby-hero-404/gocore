package env

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/spf13/viper"
)

// Load reads configuration from files and environment variables into target.
// It automatically binds nested struct keys so environment variables override file settings.
// It also performs fail-fast validation for fields marked with `required:"true"` or `validate:"required"`.
func Load(prefix string, target any, opts ...Option) error {
	if target == nil {
		return stderr.NewBadRequest("configuration target cannot be nil")
	}

	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer || val.Elem().Kind() != reflect.Struct {
		return stderr.NewBadRequest("configuration target must be a pointer to a struct")
	}

	o := &options{
		prefix: prefix,
	}
	for _, opt := range opts {
		opt(o)
	}

	// 1. Optionally parse .env file
	if o.envFile != "" {
		if err := loadDotEnv(o.envFile); err != nil && !os.IsNotExist(err) {
			return stderr.Wrap(stderr.CodeInvalidParam, err, fmt.Sprintf("failed to read env file: %s", o.envFile))
		}
	}

	v := viper.New()

	if o.prefix != "" {
		v.SetEnvPrefix(strings.ToUpper(o.prefix))
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// 2. Configure file sources if provided
	if o.configFile != "" {
		v.SetConfigFile(o.configFile)
	} else {
		if o.configName != "" {
			v.SetConfigName(o.configName)
		}
		if o.configType != "" {
			v.SetConfigType(o.configType)
		}
		for _, p := range o.configPaths {
			v.AddConfigPath(p)
		}
	}

	// 3. Bind all struct keys recursively so Viper recognizes env vars even without a file
	bindStructKeys(v, nil, val.Elem().Type())

	// 4. Read config file if specified/found
	if o.configFile != "" || (o.configName != "" && len(o.configPaths) > 0) {
		if err := v.ReadInConfig(); err != nil {
			// If a specific config file was requested and not found, report error
			if o.configFile != "" {
				return stderr.Wrap(stderr.CodeInvalidParam, err, fmt.Sprintf("failed to read config file: %s", o.configFile))
			}
			// If looking by path and not found, only report if it wasn't a ConfigFileNotFoundError
			var notFound viper.ConfigFileNotFoundError
			if !errors.As(err, &notFound) {
				return stderr.Wrap(stderr.CodeInvalidParam, err, "failed to parse config file")
			}
		}
	}

	// 5. Unmarshal into target
	if err := v.Unmarshal(target); err != nil {
		return stderr.Wrap(stderr.CodeInvalidParam, err, "failed to unmarshal configuration")
	}

	// 6. Validate required fields
	var validationErrors []stderr.FieldError
	validateRequired(val.Elem(), "", &validationErrors)

	if len(validationErrors) > 0 {
		return stderr.NewValidation("configuration validation failed", validationErrors...).
			WithKey("config_validation_failed")
	}

	return nil
}

func bindStructKeys(v *viper.Viper, parentParts []string, t reflect.Type) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		key := fieldTagKey(field)
		if key == "-" {
			continue
		}

		parts := append(parentParts, key)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			bindStructKeys(v, parts, fieldType)
		} else {
			fullKey := strings.Join(parts, ".")
			_ = v.BindEnv(fullKey)
		}
	}
}
