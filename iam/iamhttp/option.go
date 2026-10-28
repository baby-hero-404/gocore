package iamhttp

// Option configures JWT.
type Option func(*options)

type options struct {
	cookieName string
}

// WithCookie makes JWT read the token from the named cookie first (for example an HttpOnly session
// cookie set by a web app). When the cookie is absent or empty it falls back to the
// "Authorization: Bearer" header.
//
// Security note: gocore provides no CSRF protection. Services that authenticate with cookies must
// add their own (SameSite cookies, anti-CSRF tokens, or Origin/Sec-Fetch-Site checks).
func WithCookie(name string) Option {
	return func(o *options) { o.cookieName = name }
}
