package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/alexferl/zerohttp/middleware/jwtauth"

	"github.com/alexferl/zerohttp-example/store"
)

// Handler holds all HTTP handlers
type Handler struct {
	store  store.Store
	jwtCfg jwtauth.Config
}

// New creates a new Handler
func New(store store.Store, jwtCfg jwtauth.Config) *Handler {
	return &Handler{store: store, jwtCfg: jwtCfg}
}

// generateSessionID creates a unique session ID
func generateSessionID() string {
	id := make([]byte, 32) // 256 bits
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		panic("failed to generate session id")
	}
	return base64.RawURLEncoding.EncodeToString(id)
}
