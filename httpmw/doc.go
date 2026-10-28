// Package httpmw provides framework-neutral net/http middleware: request ID, panic recovery,
// request logging and CORS. A Middleware is a plain func(http.Handler) http.Handler, so it plugs
// into Chi (r.Use), http.ServeMux (wrap the mux) and any router built on net/http.
//
// The Gin equivalents in package httpserver share the security-sensitive parts with this package
// (CORS origin decision, request-ID validation), so both stacks behave the same.
package httpmw
