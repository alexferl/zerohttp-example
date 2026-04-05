package handlers

import (
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/pagination"

	"github.com/alexferl/zerohttp-example/store"

	"github.com/alexferl/zerohttp-example/models"
)

// CreateUserRequest represents a user registration request
type CreateUserRequest struct {
	Email    string          `json:"email" validate:"required,email,lowercase"`
	Name     string          `json:"name" validate:"required,min=2,max=100"`
	Password string          `json:"password" validate:"required,min=12"`
	Address  *models.Address `json:"address,omitempty"`
}

// UserResponse represents a user response
type UserResponse struct {
	ID        string          `json:"id" validate:"required"`
	Email     string          `json:"email" validate:"required,email,lowercase"`
	Name      string          `json:"name" validate:"required"`
	Address   *models.Address `json:"address,omitempty"`
	Active    bool            `json:"active"`
	Role      models.Role     `json:"role" validate:"required"`
	CreatedAt string          `json:"created_at" validate:"required"`
	UpdatedAt string          `json:"updated_at" validate:"required"`
}

// newUserResponse creates a UserResponse from a User model
func newUserResponse(user *models.User) UserResponse {
	var addr *models.Address
	if user.Address != (models.Address{}) {
		addr = &user.Address
	}

	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Address:   addr,
		Active:    user.Active,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}

// CreateUser handles POST /users
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req CreateUserRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	_, ok, err := h.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if ok {
		problem := zh.NewProblemDetail(http.StatusConflict, "user with this email already exists")
		return zh.Render.ProblemDetail(w, problem)
	}

	var addr models.Address
	if req.Address != nil {
		addr = *req.Address
	}
	user, err := models.NewUser(models.UserParams{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
		Address:  addr,
		Role:     models.RoleUser,
	})
	if err != nil {
		problem := zh.NewProblemDetail(http.StatusInternalServerError, "failed to create user")
		return zh.Render.ProblemDetail(w, problem)
	}

	if err := h.store.SaveUser(ctx, user); err != nil {
		return err
	}

	return zh.RenderAndValidate(w, http.StatusCreated, newUserResponse(user))
}

// GetUser handles GET /users/{id}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == "" {
		problem := zh.NewProblemDetail(http.StatusBadRequest, "user id is required")
		return zh.Render.ProblemDetail(w, problem)
	}

	user, ok, err := h.store.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		problem := zh.NewProblemDetail(http.StatusNotFound, "user not found")
		return zh.Render.ProblemDetail(w, problem)
	}

	return zh.RenderAndValidate(w, http.StatusOK, newUserResponse(user))
}

// UpdateUserRequest represents a profile update request
type UpdateUserRequest struct {
	Name    string          `json:"name" validate:"omitempty,min=2,max=100"`
	Address *models.Address `json:"address,omitempty"`
}

// UpdateUser handles PATCH /users/{id}
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == "" {
		problem := zh.NewProblemDetail(http.StatusBadRequest, "user id is required")
		return zh.Render.ProblemDetail(w, problem)
	}

	user, ok, err := h.store.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		problem := zh.NewProblemDetail(http.StatusNotFound, "user not found")
		return zh.Render.ProblemDetail(w, problem)
	}

	var req UpdateUserRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	if req.Address != nil {
		user.Update(req.Name, *req.Address)
	} else {
		user.Update(req.Name, user.Address)
	}

	if err := h.store.SaveUser(ctx, user); err != nil {
		return err
	}

	return zh.RenderAndValidate(w, http.StatusOK, newUserResponse(user))
}

// DeactivateUser handles POST /users/{id}/deactivate
func (h *Handler) DeactivateUser(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == "" {
		problem := zh.NewProblemDetail(http.StatusBadRequest, "user id is required")
		return zh.Render.ProblemDetail(w, problem)
	}

	user, ok, err := h.store.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		problem := zh.NewProblemDetail(http.StatusNotFound, "user not found")
		return zh.Render.ProblemDetail(w, problem)
	}

	user.Deactivate()

	if err := h.store.SaveUser(ctx, user); err != nil {
		return err
	}

	return zh.R.NoContent(w)
}

// ListUsersResponse represents a list of users response
type ListUsersResponse struct {
	Items []UserResponse `json:"items"`
}

// ListUsers handles GET /users (admin only)
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	// For now, use default pagination - could add query params later
	users, total, err := h.store.GetAllUsers(ctx, store.UserFilter{
		BaseFilter: store.BaseFilter{
			Page:    1,
			PerPage: 20,
		},
	})
	if err != nil {
		return err
	}

	params := pagination.Params{
		Page:    1,
		PerPage: 20,
	}.Defaults()

	params.WriteHeaders(w, r.URL, total)

	response := make([]UserResponse, len(users))
	for i, user := range users {
		response[i] = newUserResponse(user)
	}

	return zh.RenderAndValidate(w, http.StatusOK, ListUsersResponse{
		Items: response,
	})
}
