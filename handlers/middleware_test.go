package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name           string
		claims         map[string]any
		wantStatusCode int
		wantNextCalled bool
	}{
		{
			name:           "unauthenticated - no subject",
			claims:         map[string]any{},
			wantStatusCode: http.StatusUnauthorized,
			wantNextCalled: false,
		},
		{
			name:           "non-admin user - access denied",
			claims:         map[string]any{"sub": "user123", "scope": "read write"},
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "admin user - access granted",
			claims:         map[string]any{"sub": "admin123", "scope": "read write admin"},
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "admin scope as slice - access granted",
			claims:         map[string]any{"sub": "admin456", "scope": []string{"read", "admin"}},
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := RequireAdmin()

			nextCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tt.claims != nil {
				ctx := context.WithValue(req.Context(), jwtauth.ClaimsContextKey, tt.claims)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			zhtest.AssertEqual(t, tt.wantStatusCode, w.Code)
			zhtest.AssertEqual(t, tt.wantNextCalled, nextCalled)
		})
	}
}

func TestRequireOwner(t *testing.T) {
	tests := []struct {
		name            string
		claims          map[string]any
		resourceID      string
		wantStatusCode  int
		wantNextCalled  bool
		wantRewrittenID string
	}{
		{
			name:           "unauthenticated - no subject",
			claims:         map[string]any{},
			resourceID:     "user123",
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:            "owner matches resource ID - access granted",
			claims:          map[string]any{"sub": "user123"},
			resourceID:      "user123",
			wantStatusCode:  http.StatusOK,
			wantNextCalled:  true,
			wantRewrittenID: "user123",
		},
		{
			name:           "owner does not match - access denied",
			claims:         map[string]any{"sub": "user123"},
			resourceID:     "user456",
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:            "non-owner but admin - access granted",
			claims:          map[string]any{"sub": "admin123", "scope": "read admin"},
			resourceID:      "user456",
			wantStatusCode:  http.StatusOK,
			wantNextCalled:  true,
			wantRewrittenID: "user456",
		},
		{
			name:           "empty resource ID with owner subject - passed to next handler",
			claims:         map[string]any{"sub": "user123"},
			resourceID:     "",
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:            "me alias - rewritten to user ID",
			claims:          map[string]any{"sub": "user123"},
			resourceID:      "me",
			wantStatusCode:  http.StatusOK,
			wantNextCalled:  true,
			wantRewrittenID: "user123",
		},
		{
			name:            "me alias with admin - rewritten to admin ID",
			claims:          map[string]any{"sub": "admin123", "scope": "read admin"},
			resourceID:      "me",
			wantStatusCode:  http.StatusOK,
			wantNextCalled:  true,
			wantRewrittenID: "admin123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := RequireOwner()

			nextCalled := false
			var capturedID string
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				capturedID = r.PathValue("id")
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.resourceID, nil)
			req.SetPathValue("id", tt.resourceID)
			if tt.claims != nil {
				ctx := context.WithValue(req.Context(), jwtauth.ClaimsContextKey, tt.claims)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			zhtest.AssertEqual(t, tt.wantStatusCode, w.Code)
			zhtest.AssertEqual(t, tt.wantNextCalled, nextCalled)
			if tt.wantRewrittenID != "" {
				zhtest.AssertEqual(t, tt.wantRewrittenID, capturedID)
			}
		})
	}
}
