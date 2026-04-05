package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/alexferl/zerohttp-example/models"
	"github.com/alexferl/zerohttp-example/store"
)

func TestCreateUser_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetUserByEmail", mock.Anything, "newuser@example.com").Return(nil, false, nil)
	mockStore.On("SaveUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithJSON(map[string]any{
			"email":    "newuser@example.com",
			"name":     "New User",
			"password": "securepassword123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusCreated)

	var resp UserResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertNotEmpty(t, resp.ID)
	zhtest.AssertEqual(t, "newuser@example.com", resp.Email)
	zhtest.AssertEqual(t, "New User", resp.Name)
	zhtest.AssertEqual(t, models.RoleUser, resp.Role)
	zhtest.AssertTrue(t, resp.Active)
}

func TestCreateUser_WithAddress(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetUserByEmail", mock.Anything, "newuser@example.com").Return(nil, false, nil)
	mockStore.On("SaveUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithJSON(map[string]any{
			"email":    "newuser@example.com",
			"name":     "New User",
			"password": "securepassword123",
			"address": map[string]string{
				"street":   "123 Main St",
				"city":     "San Francisco",
				"state":    "CA",
				"zip_code": "94102",
				"country":  "USA",
			},
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusCreated)

	var resp UserResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertNotNil(t, resp.Address)
	zhtest.AssertEqual(t, "123 Main St", resp.Address.Street)
	zhtest.AssertEqual(t, "San Francisco", resp.Address.City)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	h, mockStore := setupHandler(t)

	// Create existing user
	existingUser, _ := models.NewUser(models.UserParams{
		Email:    "existing@example.com",
		Name:     "Existing User",
		Password: "password123",
	})

	mockStore.On("GetUserByEmail", mock.Anything, "existing@example.com").Return(existingUser, true, nil)

	// Try to create user with same email
	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithJSON(map[string]any{
			"email":    "existing@example.com",
			"name":     "New User",
			"password": "securepassword123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusConflict)
}

func TestCreateUser_Validation_InvalidEmail(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithJSON(map[string]any{
			"email":    "not-an-email",
			"name":     "New User",
			"password": "securepassword123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateUser(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateUser_Validation_ShortPassword(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithJSON(map[string]any{
			"email":    "newuser@example.com",
			"name":     "New User",
			"password": "short",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateUser(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateUser_Validation_ShortName(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithJSON(map[string]any{
			"email":    "newuser@example.com",
			"name":     "A",
			"password": "securepassword123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateUser(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateUser_Validation_MissingFields(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithJSON(map[string]any{}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateUser(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateUser_GetUserByEmailError(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetUserByEmail", mock.Anything, "newuser@example.com").Return(nil, false, assert.AnError)

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithJSON(map[string]any{
			"email":    "newuser@example.com",
			"name":     "New User",
			"password": "securepassword123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateUser(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateUser_SaveUserError(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetUserByEmail", mock.Anything, "newuser@example.com").Return(nil, false, nil)
	mockStore.On("SaveUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(assert.AnError)

	req := zhtest.NewRequest(http.MethodPost, "/users").
		WithJSON(map[string]any{
			"email":    "newuser@example.com",
			"name":     "New User",
			"password": "securepassword123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateUser(w, req)
	zhtest.AssertError(t, err)
}

func TestGetUser_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	mockStore.On("GetUser", mock.Anything, user.ID).Return(user, true, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+user.ID, nil)
	req.SetPathValue("id", user.ID)
	w := httptest.NewRecorder()

	err := h.GetUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp UserResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, user.ID, resp.ID)
	zhtest.AssertEqual(t, user.Email, resp.Email)
	zhtest.AssertEqual(t, user.Name, resp.Name)
}

func TestGetUser_NotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetUser", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/nonexistent-id", nil)
	req.SetPathValue("id", "nonexistent-id")
	w := httptest.NewRecorder()

	err := h.GetUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestGetUser_MissingID(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	err := h.GetUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusBadRequest)
}

func TestUpdateUser_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	mockStore.On("GetUser", mock.Anything, user.ID).Return(user, true, nil)
	mockStore.On("SaveUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

	req := zhtest.NewRequest(http.MethodPatch, "/users/"+user.ID).
		WithJSON(map[string]any{
			"name": "Updated Name",
		}).
		Build()
	req.SetPathValue("id", user.ID)
	w := httptest.NewRecorder()

	err := h.UpdateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp UserResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, "Updated Name", resp.Name)
	zhtest.AssertEqual(t, user.Email, resp.Email) // Email unchanged
}

func TestUpdateUser_WithAddress(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	mockStore.On("GetUser", mock.Anything, user.ID).Return(user, true, nil)
	mockStore.On("SaveUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

	req := zhtest.NewRequest(http.MethodPatch, "/users/"+user.ID).
		WithJSON(map[string]any{
			"name": "Updated Name",
			"address": map[string]string{
				"street":   "456 Oak St",
				"city":     "Los Angeles",
				"state":    "CA",
				"zip_code": "90210",
				"country":  "USA",
			},
		}).
		Build()
	req.SetPathValue("id", user.ID)
	w := httptest.NewRecorder()

	err := h.UpdateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp UserResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, "Updated Name", resp.Name)
	zhtest.AssertNotNil(t, resp.Address)
	zhtest.AssertEqual(t, "456 Oak St", resp.Address.Street)
	zhtest.AssertEqual(t, "Los Angeles", resp.Address.City)
}

func TestUpdateUser_NotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetUser", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := zhtest.NewRequest(http.MethodPatch, "/users/nonexistent-id").
		WithJSON(map[string]any{
			"name": "Updated Name",
		}).
		Build()
	req.SetPathValue("id", "nonexistent-id")
	w := httptest.NewRecorder()

	err := h.UpdateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestUpdateUser_MissingID(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPatch, "/users/").
		WithJSON(map[string]any{
			"name": "Updated Name",
		}).
		Build()
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	err := h.UpdateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusBadRequest)
}

func TestUpdateUser_Validation_ShortName(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	mockStore.On("GetUser", mock.Anything, user.ID).Return(user, true, nil)

	req := zhtest.NewRequest(http.MethodPatch, "/users/"+user.ID).
		WithJSON(map[string]any{
			"name": "A",
		}).
		Build()
	req.SetPathValue("id", user.ID)
	w := httptest.NewRecorder()

	err := h.UpdateUser(w, req)
	zhtest.AssertError(t, err)
}

func TestDeactivateUser_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	// Verify user starts active
	zhtest.AssertTrue(t, user.Active)

	mockStore.On("GetUser", mock.Anything, user.ID).Return(user, true, nil)
	mockStore.On("SaveUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/users/"+user.ID+"/deactivate", nil)
	req.SetPathValue("id", user.ID)
	w := httptest.NewRecorder()

	err := h.DeactivateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusNoContent)
	zhtest.AssertWith(t, w).BodyEmpty()
}

func TestDeactivateUser_NotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetUser", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := httptest.NewRequest(http.MethodPost, "/users/nonexistent-id/deactivate", nil)
	req.SetPathValue("id", "nonexistent-id")
	w := httptest.NewRecorder()

	err := h.DeactivateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestDeactivateUser_MissingID(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/users//deactivate", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	err := h.DeactivateUser(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusBadRequest)
}

func TestListUsers_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	// Create multiple users
	users := []*models.User{
		func() *models.User {
			u, _ := models.NewUser(models.UserParams{
				Email:    "user0@example.com",
				Name:     "User 0",
				Password: "password123",
			})
			return u
		}(),
		func() *models.User {
			u, _ := models.NewUser(models.UserParams{
				Email:    "user1@example.com",
				Name:     "User 1",
				Password: "password123",
			})
			return u
		}(),
		func() *models.User {
			u, _ := models.NewUser(models.UserParams{
				Email:    "user2@example.com",
				Name:     "User 2",
				Password: "password123",
			})
			return u
		}(),
	}

	mockStore.On("GetAllUsers", mock.Anything, store.UserFilter{BaseFilter: store.BaseFilter{Page: 1, PerPage: 20}}).Return(users, 3, nil)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	err := h.ListUsers(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListUsersResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertLen(t, resp.Items, 3)
	zhtest.AssertEqual(t, "3", w.Header().Get(httpx.HeaderXTotal))
}

func TestListUsers_Empty(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetAllUsers", mock.Anything, store.UserFilter{BaseFilter: store.BaseFilter{Page: 1, PerPage: 20}}).Return([]*models.User{}, 0, nil)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	err := h.ListUsers(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListUsersResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertLen(t, resp.Items, 0)
	zhtest.AssertEqual(t, "0", w.Header().Get(httpx.HeaderXTotal))
}
