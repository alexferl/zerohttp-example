package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/stretchr/testify/mock"

	"github.com/alexferl/zerohttp-example/models"
	"github.com/alexferl/zerohttp-example/store"
)

// Helper to create context with user info (simulating JWT middleware)
func withUserContext(ctx context.Context, userID string, role models.Role) context.Context {
	ctx = context.WithValue(ctx, jwtauth.ClaimsContextKey, map[string]any{
		"sub":   userID,
		"scope": string(role),
	})
	return ctx
}

func TestCreateOrder_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	// Create a record to order
	record := models.NewRecord(models.RecordParams{
		Title:     "Test Record",
		Artist:    "Test Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreJazz,
		Condition: models.ConditionNM,
		Price:     25.00,
		Stock:     10,
	})

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)
	mockStore.On("DecrementStock", mock.Anything, record.ID, 2).Return(true, nil)
	mockStore.On("SaveOrder", mock.Anything, mock.AnythingOfType("*models.Order")).Return(nil)

	// Create authenticated request
	// Note: Idempotency is handled by middleware in production, not the handler
	req := zhtest.NewRequest(http.MethodPost, "/orders").
		WithJSON(map[string]any{
			"items": []map[string]any{
				{
					"record_id": record.ID,
					"quantity":  2,
				},
			},
			"shipping_address": map[string]string{
				"street":   "123 Main St",
				"city":     "SF",
				"state":    "CA",
				"zip_code": "94102",
				"country":  "USA",
			},
		}).
		Build()
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.CreateOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusCreated)

	var resp OrderResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertNotEmpty(t, resp.ID)
	zhtest.AssertEqual(t, user.ID, resp.UserID)
	zhtest.AssertEqual(t, "pending", resp.Status)
	zhtest.AssertEqual(t, 50.00, resp.Total) // 2 * $25.00
	zhtest.AssertLen(t, resp.Items, 1)
}

func TestCreateOrder_Unauthorized(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/orders").
		WithJSON(map[string]any{
			"items": []map[string]any{
				{
					"record_id": "some-id",
					"quantity":  1,
				},
			},
		}).
		Build()
	// No auth context
	w := httptest.NewRecorder()

	err := h.CreateOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusUnauthorized)
}

func TestCreateOrder_RecordNotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	mockStore.On("GetRecord", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := zhtest.NewRequest(http.MethodPost, "/orders").
		WithJSON(map[string]any{
			"items": []map[string]any{
				{
					"record_id": "nonexistent-id",
					"quantity":  1,
				},
			},
		}).
		Build()
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.CreateOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	record := models.NewRecord(models.RecordParams{
		Title:     "Limited Stock",
		Artist:    "Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreJazz,
		Condition: models.ConditionNM,
		Price:     25.00,
		Stock:     2,
	})

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)
	mockStore.On("DecrementStock", mock.Anything, record.ID, 5).Return(false, nil)

	req := zhtest.NewRequest(http.MethodPost, "/orders").
		WithJSON(map[string]any{
			"items": []map[string]any{
				{
					"record_id": record.ID,
					"quantity":  5, // More than available
				},
			},
		}).
		Build()
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.CreateOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusConflict)
}

func TestCreateOrder_ArchivedRecord(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	record := models.NewRecord(models.RecordParams{
		Title:     "Archived Record",
		Artist:    "Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreJazz,
		Condition: models.ConditionNM,
		Price:     25.00,
		Stock:     10,
	})
	record.Archived = true

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)

	req := zhtest.NewRequest(http.MethodPost, "/orders").
		WithJSON(map[string]any{
			"items": []map[string]any{
				{
					"record_id": record.ID,
					"quantity":  1,
				},
			},
		}).
		Build()
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.CreateOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusConflict)
}

func TestCreateOrder_MultipleItems(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	record1 := models.NewRecord(models.RecordParams{
		Title:     "Record 1",
		Artist:    "Artist 1",
		Format:    models.FormatLP,
		Genre:     models.GenreJazz,
		Condition: models.ConditionNM,
		Price:     20.00,
		Stock:     10,
	})
	record2 := models.NewRecord(models.RecordParams{
		Title:     "Record 2",
		Artist:    "Artist 2",
		Format:    models.FormatLP,
		Genre:     models.GenreRock,
		Condition: models.ConditionVGPlus,
		Price:     30.00,
		Stock:     5,
	})

	mockStore.On("GetRecord", mock.Anything, record1.ID).Return(record1, true, nil)
	mockStore.On("GetRecord", mock.Anything, record2.ID).Return(record2, true, nil)
	mockStore.On("DecrementStock", mock.Anything, record1.ID, 2).Return(true, nil)
	mockStore.On("DecrementStock", mock.Anything, record2.ID, 1).Return(true, nil)
	mockStore.On("SaveOrder", mock.Anything, mock.AnythingOfType("*models.Order")).Return(nil)

	req := zhtest.NewRequest(http.MethodPost, "/orders").
		WithJSON(map[string]any{
			"items": []map[string]any{
				{
					"record_id": record1.ID,
					"quantity":  2,
				},
				{
					"record_id": record2.ID,
					"quantity":  1,
				},
			},
		}).
		Build()
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.CreateOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusCreated)

	var resp OrderResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, 70.00, resp.Total) // (2 * $20) + (1 * $30)
	zhtest.AssertLen(t, resp.Items, 2)
}

func TestListOrders_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	// Create orders for the user
	orders := []*models.Order{
		models.NewOrder(models.OrderParams{
			UserID: user.ID,
			Items: []models.OrderItem{
				{RecordID: "record-0", Title: "Record A", Artist: "Artist", Price: 25.00, Quantity: 1},
			},
		}),
		models.NewOrder(models.OrderParams{
			UserID: user.ID,
			Items: []models.OrderItem{
				{RecordID: "record-1", Title: "Record B", Artist: "Artist", Price: 25.00, Quantity: 1},
			},
		}),
		models.NewOrder(models.OrderParams{
			UserID: user.ID,
			Items: []models.OrderItem{
				{RecordID: "record-2", Title: "Record C", Artist: "Artist", Price: 25.00, Quantity: 1},
			},
		}),
	}

	mockStore.On("GetOrdersByUser", mock.Anything, store.OrderFilter{BaseFilter: store.BaseFilter{Page: 0, PerPage: 0}, UserID: user.ID}).Return(orders, 3, nil)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.ListOrders(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListOrdersResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertLen(t, resp.Items, 3)
	zhtest.AssertEqual(t, "3", w.Header().Get(httpx.HeaderXTotal))
}

func TestListOrders_FilterByStatus(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	// Create orders with different statuses
	pendingOrder := models.NewOrder(models.OrderParams{
		UserID: user.ID,
		Items:  []models.OrderItem{{RecordID: "r1", Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 1}},
	})
	pendingOrder.Status = models.OrderStatusPending

	shippedOrder := models.NewOrder(models.OrderParams{
		UserID: user.ID,
		Items:  []models.OrderItem{{RecordID: "r2", Title: "Record 2", Artist: "Artist", Price: 30.00, Quantity: 1}},
	})
	shippedOrder.Status = models.OrderStatusShipped

	mockStore.On("GetOrdersByUser", mock.Anything, store.OrderFilter{BaseFilter: store.BaseFilter{Page: 0, PerPage: 0}, UserID: user.ID, Status: "pending"}).Return([]*models.Order{pendingOrder}, 1, nil)

	// Filter by pending status
	req := httptest.NewRequest(http.MethodGet, "/orders?status=pending", nil)
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.ListOrders(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListOrdersResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, "1", w.Header().Get(httpx.HeaderXTotal))
	zhtest.AssertEqual(t, "pending", resp.Items[0].Status)
}

func TestListOrders_Unauthorized(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	// No auth context
	w := httptest.NewRecorder()

	err := h.ListOrders(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusUnauthorized)
}

func TestGetOrder_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	order := models.NewOrder(models.OrderParams{
		UserID: user.ID,
		Items: []models.OrderItem{
			{RecordID: "r1", Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 1},
		},
	})

	mockStore.On("GetOrder", mock.Anything, order.ID).Return(order, true, nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+order.ID, nil)
	req.SetPathValue("id", order.ID)
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.GetOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp OrderResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, order.ID, resp.ID)
	zhtest.AssertEqual(t, user.ID, resp.UserID)
}

func TestGetOrder_NotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	mockStore.On("GetOrder", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/nonexistent-id", nil)
	req.SetPathValue("id", "nonexistent-id")
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.GetOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestGetOrder_AccessDenied(t *testing.T) {
	h, mockStore := setupHandler(t)

	user1, _ := models.NewUser(models.UserParams{
		Email:    "user1@example.com",
		Name:     "User 1",
		Password: "password123",
		Role:     models.RoleUser,
	})

	// Create another user
	user2, _ := models.NewUser(models.UserParams{
		Email:    "user2@example.com",
		Name:     "User 2",
		Password: "password123",
		Role:     models.RoleUser,
	})

	// Create order for user2
	order := models.NewOrder(models.OrderParams{
		UserID: user2.ID,
		Items: []models.OrderItem{
			{RecordID: "r1", Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 1},
		},
	})

	mockStore.On("GetOrder", mock.Anything, order.ID).Return(order, true, nil)

	// Try to access as user1
	req := httptest.NewRequest(http.MethodGet, "/orders/"+order.ID, nil)
	req.SetPathValue("id", order.ID)
	req = req.WithContext(withUserContext(req.Context(), user1.ID, user1.Role))
	w := httptest.NewRecorder()

	err := h.GetOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusForbidden)
}

func TestGetOrder_AdminCanAccessAnyOrder(t *testing.T) {
	h, mockStore := setupHandler(t)

	// Create admin user
	admin, _ := models.NewUser(models.UserParams{
		Email:    "admin@example.com",
		Name:     "Admin",
		Password: "admin123",
		Role:     models.RoleAdmin,
	})

	// Create a regular user
	user, _ := models.NewUser(models.UserParams{
		Email:    "user@example.com",
		Name:     "User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	// Create order for regular user
	order := models.NewOrder(models.OrderParams{
		UserID: user.ID,
		Items: []models.OrderItem{
			{RecordID: "r1", Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 1},
		},
	})

	mockStore.On("GetOrder", mock.Anything, order.ID).Return(order, true, nil)

	// Access as admin
	req := httptest.NewRequest(http.MethodGet, "/orders/"+order.ID, nil)
	req.SetPathValue("id", order.ID)
	req = req.WithContext(withUserContext(req.Context(), admin.ID, admin.Role))
	w := httptest.NewRecorder()

	err := h.GetOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
}

func TestCancelOrder_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	record := models.NewRecord(models.RecordParams{
		Title:     "Test Record",
		Artist:    "Test Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreJazz,
		Condition: models.ConditionNM,
		Price:     25.00,
		Stock:     10,
	})

	order := models.NewOrder(models.OrderParams{
		UserID: user.ID,
		Items: []models.OrderItem{
			{RecordID: record.ID, Title: record.Title, Artist: record.Artist, Price: record.Price, Quantity: 2},
		},
	})

	// Verify initial stock
	zhtest.AssertEqual(t, 10, record.Stock)

	mockStore.On("GetOrder", mock.Anything, order.ID).Return(order, true, nil)
	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)
	mockStore.On("SaveRecord", mock.Anything, mock.AnythingOfType("*models.Record")).Return(nil)
	mockStore.On("SaveOrder", mock.Anything, mock.AnythingOfType("*models.Order")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/orders/"+order.ID+"/cancel", nil)
	req.SetPathValue("id", order.ID)
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.CancelOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusNoContent)
}

func TestCancelOrder_NotPending(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	order := models.NewOrder(models.OrderParams{
		UserID: user.ID,
		Items: []models.OrderItem{
			{RecordID: "r1", Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 1},
		},
	})
	order.Status = models.OrderStatusShipped

	mockStore.On("GetOrder", mock.Anything, order.ID).Return(order, true, nil)

	req := httptest.NewRequest(http.MethodPost, "/orders/"+order.ID+"/cancel", nil)
	req.SetPathValue("id", order.ID)
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.CancelOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusConflict)
}

func TestCancelOrder_NotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	user, _ := models.NewUser(models.UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     models.RoleUser,
	})

	mockStore.On("GetOrder", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := httptest.NewRequest(http.MethodPost, "/orders/nonexistent-id/cancel", nil)
	req.SetPathValue("id", "nonexistent-id")
	req = req.WithContext(withUserContext(req.Context(), user.ID, user.Role))
	w := httptest.NewRecorder()

	err := h.CancelOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestCancelOrder_AccessDenied(t *testing.T) {
	h, mockStore := setupHandler(t)

	user1, _ := models.NewUser(models.UserParams{
		Email:    "user1@example.com",
		Name:     "User 1",
		Password: "password123",
		Role:     models.RoleUser,
	})

	// Create another user
	user2, _ := models.NewUser(models.UserParams{
		Email:    "user2@example.com",
		Name:     "User 2",
		Password: "password123",
		Role:     models.RoleUser,
	})

	// Create order for user2
	order := models.NewOrder(models.OrderParams{
		UserID: user2.ID,
		Items: []models.OrderItem{
			{RecordID: "r1", Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 1},
		},
	})

	mockStore.On("GetOrder", mock.Anything, order.ID).Return(order, true, nil)

	// Try to cancel as user1
	req := httptest.NewRequest(http.MethodPost, "/orders/"+order.ID+"/cancel", nil)
	req.SetPathValue("id", order.ID)
	req = req.WithContext(withUserContext(req.Context(), user1.ID, user1.Role))
	w := httptest.NewRecorder()

	err := h.CancelOrder(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusForbidden)
}
