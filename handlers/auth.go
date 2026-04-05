package handlers

import (
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/jwtauth"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token" validate:"required"`
	RefreshToken string `json:"refresh_token" validate:"required"`
	TokenType    string `json:"token_type" validate:"required,eq=Bearer"`
	ExpiresIn    int    `json:"expires_in" validate:"required"`
}

// Login handles POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req LoginRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	user, ok, err := h.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if !ok || !user.VerifyPassword(req.Password) {
		problem := zh.NewProblemDetail(http.StatusUnauthorized, "invalid credentials")
		return zh.Render.ProblemDetail(w, problem)
	}

	if !user.Active {
		problem := zh.NewProblemDetail(http.StatusForbidden, "account deactivated")
		return zh.Render.ProblemDetail(w, problem)
	}

	sessionID := generateSessionID()

	accessClaims := map[string]any{
		"sub":   user.ID,
		"scope": string(user.Role),
		"jti":   generateSessionID(),
		"sid":   sessionID,
	}

	accessToken, err := jwtauth.GenerateAccessToken(r, accessClaims, h.jwtCfg)
	if err != nil {
		problem := zh.NewProblemDetail(http.StatusInternalServerError, "failed to generate access token")
		return zh.Render.ProblemDetail(w, problem)
	}

	refreshClaims := map[string]any{
		"sub":   user.ID,
		"scope": string(user.Role),
		"jti":   generateSessionID(),
		"sid":   sessionID,
		"type":  "refresh",
	}

	refreshToken, err := jwtauth.GenerateRefreshToken(r, refreshClaims, h.jwtCfg)
	if err != nil {
		problem := zh.NewProblemDetail(http.StatusInternalServerError, "failed to generate refresh token")
		return zh.Render.ProblemDetail(w, problem)
	}

	resp := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.jwtCfg.AccessTokenTTL.Seconds()),
	}

	return zh.RenderAndValidate(w, http.StatusOK, resp)
}
