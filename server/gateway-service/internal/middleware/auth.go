// Package middleware provides HTTP middleware for the gateway service.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

// keycloakClaims is the subset of Keycloak JWT claims used for authorisation.
type keycloakClaims struct {
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

type claimsContextKey struct{}

// Verifier holds the OIDC token verifier.
// When the inner verifier is nil the service runs in permissive mode —
// all requests pass through without authentication.
// Permissive mode is activated when KEYCLOAK_URL is not set.
type Verifier struct {
	v *oidc.IDTokenVerifier
}

// NewVerifier fetches Keycloak's OIDC provider metadata and returns a Verifier.
// If keycloakURL is empty it returns a permissive Verifier (dev mode, no error).
func NewVerifier(ctx context.Context, keycloakURL, realm string) (*Verifier, error) {
	if keycloakURL == "" {
		return &Verifier{}, nil
	}
	issuer := strings.TrimRight(keycloakURL, "/") + "/realms/" + realm
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc: connect to keycloak at %s: %w", issuer, err)
	}
	return &Verifier{
		v: provider.Verifier(&oidc.Config{SkipClientIDCheck: true, SkipIssuerCheck: true}),
	}, nil
}

// Permissive reports whether the verifier is in permissive (no-auth) mode.
func (v *Verifier) Permissive() bool { return v.v == nil }

// RequireAuth is a chi-compatible middleware that validates the Bearer JWT in
// the Authorization header.  On success it stores the parsed claims in the
// request context so downstream middleware (e.g. RequireRole) can read them.
// In permissive mode every request passes through.
func (v *Verifier) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v.v == nil {
			next.ServeHTTP(w, r)
			return
		}
		raw := bearerToken(r)
		if raw == "" {
			authError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid token")
			return
		}
		idToken, err := v.v.Verify(r.Context(), raw)
		if err != nil {
			authError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid token")
			return
		}
		var c keycloakClaims
		if err := idToken.Claims(&c); err != nil {
			authError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsContextKey{}, &c)))
	})
}

// RequireRole returns a chi-compatible middleware that checks the
// realm_access.roles claim for the given role.  It must be used after
// RequireAuth.  In permissive mode every request passes through.
func (v *Verifier) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if v.v == nil {
				next.ServeHTTP(w, r)
				return
			}
			c, _ := r.Context().Value(claimsContextKey{}).(*keycloakClaims)
			if c == nil || !containsRole(c.RealmAccess.Roles, role) {
				authError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken extracts the raw JWT from the Authorization: Bearer <token> header.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return h[len("Bearer "):]
}

func containsRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

// authError writes a JSON error response consistent with the gateway's error envelope.
func authError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"code":%q,"message":%q}}`, code, message)
}
