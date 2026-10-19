package env

// Option configures the config loader.
type Option func(*options)

type options struct {
	configPaths []string
	configName  string
	configType  string
	configFile  string
	envFile     string
	prefix      string
}

// WithConfigPath adds a directory path to look for config files.
func WithConfigPath(path string) Option {
	return func(o *options) {
		o.configPaths = append(o.configPaths, path)
	}
}

// WithConfigName sets the config file name (e.g. "config").
func WithConfigName(name string) Option {
	return func(o *options) {
		o.configName = name
	}
}

// WithConfigType sets the config file type (e.g. "yaml", "json").
func WithConfigType(typ string) Option {
	return func(o *options) {
		o.configType = typ
	}
}

// WithConfigFile sets a direct path to the config file.
func WithConfigFile(file string) Option {
	return func(o *options) {
		o.configFile = file
	}
}

// WithEnvFile specifies a .env file to load into process environment first.
func WithEnvFile(file string) Option {
	return func(o *options) {
		o.envFile = file
	}
}

// WithPrefix sets the environment variable prefix (e.g. "APP" -> APP_HOST).
func WithPrefix(prefix string) Option {
	return func(o *options) {
		o.prefix = prefix
	}
}
