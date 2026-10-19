package env

// Loader defines the contract for loading configuration.
type Loader interface {
	Load(prefix string, target any, opts ...Option) error
}
