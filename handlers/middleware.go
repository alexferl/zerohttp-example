package handlers

import (
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/jwtauth"

	"github.com/alexferl/zerohttp-example/models"
)

// contextKey is a type for context keys.
// Using a struct instead of string prevents collisions with keys from other packages.
type contextKey struct{}

var (
	// ContextKeyUserID is the key for user ID in context.
	// Using a pointer ensures uniqueness across packages.
	ContextKeyUserID = &contextKey{}
	// ContextKeyRole is the key for role in context.
	ContextKeyRole = &contextKey{}
)

// RequireAdmin validates admin role claim.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := jwtauth.GetClaims(r)
			if claims.Subject() == "" {
				problem := zh.NewProblemDetail(http.StatusUnauthorized, "authentication required")
				_ = zh.Render.ProblemDetail(w, problem)
				return
			}
			if !claims.HasScope("admin") {
				problem := zh.NewProblemDetail(http.StatusForbidden, "admin access required")
				_ = zh.Render.ProblemDetail(w, problem)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireOwner validates resource ownership matches JWT sub.
// Use this when the route has a {id} parameter that should match the user ID.
func RequireOwner() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := jwtauth.GetClaims(r)
			userID := claims.Subject()
			resourceID := zh.Param(r, "id")

			// If user is admin, allow access to any resource
			if claims.HasScope("admin") {
				next.ServeHTTP(w, r)
				return
			}

			// Check if the resource ID matches the authenticated user
			if resourceID != "" && resourceID != userID {
				problem := zh.NewProblemDetail(http.StatusForbidden, "access denied")
				_ = zh.Render.ProblemDetail(w, problem)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID returns the user ID from context.
func GetUserID(r *http.Request) string {
	claims := jwtauth.GetClaims(r)
	return claims.Subject()
}

// GetRole returns the role from context.
func GetRole(r *http.Request) models.Role {
	claims := jwtauth.GetClaims(r)
	if claims.HasScope("admin") {
		return models.RoleAdmin
	}
	return models.RoleUser
}
