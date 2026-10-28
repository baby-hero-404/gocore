package iamgin

// Option configures JWTMiddleware.
type Option func(*options)

type options struct {
	cookieName string
}

// WithCookie makes JWTMiddleware read the token from the named cookie first
// (e.g. an HttpOnly session cookie set by a web app). When the cookie is absent
// or empty, the middleware falls back to the "Authorization: Bearer" header.
//
// Security Note: gocore does not provide built-in CSRF protection. Consuming services
// that use cookie-based authentication must implement their own CSRF defenses
// (e.g. SameSite=Lax/Strict cookies, anti-CSRF tokens, or origin/Sec-Fetch-Site verification).
func WithCookie(name string) Option {
	return func(o *options) {
		o.cookieName = name
	}
}
