// Package auth resolves the Tailscale identity of inbound HTTP callers.
//
// This package is now a thin shim: all middleware logic, WhoIs calls, IP
// parsing, loopback detection, and context injection are handled by
// tailkit.AuthMiddleware and tailkit.CallerFromContext. The public helpers
// here exist only so call sites in the rest of the codebase (e.g. callerHost
// in api/files.go) continue to compile without changes.
package auth

import (
	"context"
	"net/http"

	tailkit "github.com/wf-pro-dev/tailkit"
)

// Identity is an alias for tailkit.CallerIdentity so existing call sites that
// do auth.FromContext(ctx) and read .Hostname keep compiling unchanged.
type Identity = tailkit.CallerIdentity

// FromContext retrieves the caller identity injected by Middleware.
// Returns (zero, false) for unauthenticated requests (e.g. loopback dev).
func FromContext(ctx context.Context) (Identity, bool) {
	return tailkit.CallerFromContext(ctx)
}

// Middleware wraps tailkit.AuthMiddleware so callers import only this package.
// The *tailkit.Server provides the tsnet LocalClient used internally by
// tailkit.AuthMiddleware to call lc.WhoIs on every non-loopback request.
func Middleware(srv *tailkit.Server) func(http.Handler) http.Handler {
	return tailkit.AuthMiddleware(srv)
}
