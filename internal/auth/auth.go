package auth

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"tailscale.com/client/local"
)

type contextKey string

const identityKey contextKey = "identity"

// Identity holds the resolved Tailscale identity of the caller.
type Identity struct {
	Hostname    string
	UserLogin   string
	TailscaleIP string
}

func (id Identity) String() string {
	return fmt.Sprintf("%s (%s) @ %s", id.Hostname, id.UserLogin, id.TailscaleIP)
}

func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}

// Middleware resolves the Tailscale identity of the caller.
//
// Request flow:
//
//	browser (100.x.x.x) → nginx (172.x.x.x) → backend
//
// When a request arrives from a private IP (nginx), we trust the
// X-Forwarded-For header to get the real Tailscale IP of the browser.
// When a request arrives directly on the tailnet (100.x.x.x), we use
// RemoteAddr directly.
// Loopback requests (local dev) are allowed through with no identity.
func Middleware(lc *local.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				remoteIP = r.RemoteAddr
			}

			// Loopback — local dev, no identity needed.
			if isLoopback(remoteIP) {
				next.ServeHTTP(w, r)
				return
			}

			// Request came through nginx (private/Docker IP).
			// Trust X-Forwarded-For to get the real client IP.
			clientIP := remoteIP
			if isPrivate(remoteIP) {
				if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
					// Take the first IP in the chain — that's the original client.
					clientIP = strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
				} else {
					// No forwarded header — internal request (e.g. health check).
					// Allow through without identity.
					next.ServeHTTP(w, r)
					return
				}
			}

			addr, err := netip.ParseAddr(clientIP)
			if err != nil {
				log.Printf("auth: could not parse client IP %q: %v", clientIP, err)
				http.Error(w, "could not parse client address", http.StatusBadRequest)
				return
			}

			info, err := lc.WhoIs(r.Context(), addr.String())
			if err != nil {
				log.Printf("auth: WhoIs failed for %s: %v", addr, err)
				http.Error(w, "identity resolution failed", http.StatusUnauthorized)
				return
			}

			id := Identity{TailscaleIP: addr.String()}

			if info.Node != nil {
				dnsName := strings.TrimSuffix(info.Node.Name, ".")
				if parts := strings.SplitN(dnsName, ".", 2); len(parts) > 0 {
					id.Hostname = parts[0]
				}
			}
			if info.UserProfile != nil {
				id.UserLogin = info.UserProfile.LoginName
			}

			log.Printf("auth: request from %s", id)

			ctx := context.WithValue(r.Context(), identityKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func isLoopback(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil && ip.IsLoopback()
}

func isPrivate(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	for _, cidr := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
