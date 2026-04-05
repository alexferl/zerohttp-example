package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/alexferl/zerohttp-example/mocks"
	"github.com/alexferl/zerohttp-example/models"
)

// setupHandler creates a handler with a mock store and test JWT config for testing
func setupHandler(t *testing.T) (*Handler, *mocks.MockStore) {
	t.Helper()
	mockStore := mocks.NewMockStore(t)

	// Create a test HS256 store
	jwtSecret := []byte("test-secret-key-at-least-32-bytes!")
	tokenStore := jwtauth.NewHS256Store(jwtSecret, jwtauth.HS256Config{
		Issuer: "test-issuer",
	})

	jwtCfg := jwtauth.Config{
		Store:           tokenStore,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	h := New(mockStore, jwtCfg)
	return h, mockStore
}

// setupHandlerWithUser creates a handler with a pre-registered test user
func setupHandlerWithUser(t *testing.T) (*Handler, *mocks.MockStore, *models.User) {
	h, mockStore := setupHandler(t)

	user, err := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})
	zhtest.AssertNoError(t, err)

	mockStore.EXPECT().GetUserByEmail(mock.Anything, "test@example.com").Return(user, true, nil).Maybe()
	mockStore.EXPECT().GetUser(mock.Anything, user.ID).Return(user, true, nil).Maybe()
	mockStore.EXPECT().SaveUser(mock.Anything, mock.Anything).Return(nil).Maybe()

	return h, mockStore, user
}

func TestLogin_Success(t *testing.T) {
	h, _, _ := setupHandlerWithUser(t)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp LoginResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertNotEmpty(t, resp.AccessToken)
	zhtest.AssertNotEmpty(t, resp.RefreshToken)
	zhtest.AssertEqual(t, "Bearer", resp.TokenType)
	zhtest.AssertEqual(t, 900, resp.ExpiresIn) // 15 minutes = 900 seconds
}

func TestLogin_GetUserByEmailError(t *testing.T) {
	h, mockStore, _ := setupHandlerWithUser(t)

	// Override mock to return an error
	mockStore.ExpectedCalls = nil
	mockStore.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, false, assert.AnError)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertError(t, err)
}

func TestLogin_InvalidCredentials_WrongPassword(t *testing.T) {
	h, mockStore, user := setupHandlerWithUser(t)

	// Override the mock to return the user for this test
	mockStore.ExpectedCalls = nil
	mockStore.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, true, nil)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"email":    "test@example.com",
			"password": "wrongpassword",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusUnauthorized)
}

func TestLogin_InvalidCredentials_UserNotFound(t *testing.T) {
	h, mockStore, _ := setupHandlerWithUser(t)

	// Override mock to return not found for nonexistent user
	mockStore.ExpectedCalls = nil
	mockStore.On("GetUserByEmail", mock.Anything, "nonexistent@example.com").Return(nil, false, nil)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"email":    "nonexistent@example.com",
			"password": "password123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusUnauthorized)
}

func TestLogin_InactiveAccount(t *testing.T) {
	h, mockStore, user := setupHandlerWithUser(t)
	user.Deactivate()

	// Override mock to return inactive user
	mockStore.ExpectedCalls = nil
	mockStore.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, true, nil)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"email":    "test@example.com",
			"password": "password123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusForbidden)
}

func TestLogin_InvalidJSON(t *testing.T) {
	h, _, _ := setupHandlerWithUser(t)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithHeader("Content-Type", "application/json").
		WithBody(http.NoBody).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	// BindAndValidate returns an error for invalid JSON, which should be handled by the framework
	zhtest.AssertError(t, err)
}

func TestLogin_Validation_MissingEmail(t *testing.T) {
	h, _, _ := setupHandlerWithUser(t)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"password": "password123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertError(t, err)
}

func TestLogin_Validation_InvalidEmail(t *testing.T) {
	h, _, _ := setupHandlerWithUser(t)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"email":    "not-an-email",
			"password": "password123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertError(t, err)
}

func TestLogin_Validation_MissingPassword(t *testing.T) {
	h, _, _ := setupHandlerWithUser(t)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"email": "test@example.com",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertError(t, err)
}

func TestLogin_AdminUser_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	// Create an admin user
	admin, err := models.NewUser(models.UserParams{
		Email:    "admin@example.com",
		Name:     "Admin User",
		Password: "admin123",
		Role:     models.RoleAdmin,
	})
	zhtest.AssertNoError(t, err)

	mockStore.On("GetUserByEmail", mock.Anything, "admin@example.com").Return(admin, true, nil)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"email":    "admin@example.com",
			"password": "admin123",
		}).
		Build()
	w := httptest.NewRecorder()

	err = h.Login(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp LoginResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertNotEmpty(t, resp.AccessToken)
	zhtest.AssertNotEmpty(t, resp.RefreshToken)
}

func TestLogin_EmptyBody(t *testing.T) {
	h, _, _ := setupHandlerWithUser(t)

	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithHeader("Content-Type", "application/json").
		WithBytes([]byte{}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertError(t, err)
}

func TestLogin_CaseSensitiveEmail(t *testing.T) {
	h, mockStore, _ := setupHandlerWithUser(t)

	// Override mock to return not found for uppercase email
	mockStore.ExpectedCalls = nil
	mockStore.On("GetUserByEmail", mock.Anything, "TEST@EXAMPLE.COM").Return(nil, false, nil)

	// Try with uppercase email - validation should fail due to lowercase tag
	req := zhtest.NewRequest(http.MethodPost, "/auth/login").
		WithJSON(map[string]string{
			"email":    "TEST@EXAMPLE.COM",
			"password": "password123",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.Login(w, req)
	zhtest.AssertNoError(t, err)

	// The lowercase validator rejects uppercase emails (401 from failed lookup)
	// To support case-insensitive lookup, normalize emails before storage/lookup
	zhtest.AssertWith(t, w).Status(http.StatusUnauthorized)
}
