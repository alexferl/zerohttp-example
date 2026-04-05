package handlers

import (
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/pagination"

	"github.com/alexferl/zerohttp-example/models"
	"github.com/alexferl/zerohttp-example/store"
)

// OrderItemRequest represents an item in an order request
type OrderItemRequest struct {
	RecordID string `json:"record_id" validate:"required"`
	Quantity int    `json:"quantity" validate:"required,min=1"`
}

// CreateOrderRequest represents an order creation request
type CreateOrderRequest struct {
	Items           []OrderItemRequest `json:"items" validate:"required,min=1,each"`
	ShippingAddress models.Address     `json:"shipping_address"`
}

// OrderItemResponse represents an item in an order response
type OrderItemResponse struct {
	RecordID string  `json:"record_id"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

// OrderResponse represents an order response
type OrderResponse struct {
	ID              string              `json:"id"`
	UserID          string              `json:"user_id"`
	Items           []OrderItemResponse `json:"items"`
	Total           float64             `json:"total"`
	Status          string              `json:"status"`
	ShippingAddress models.Address      `json:"shipping_address"`
	CreatedAt       string              `json:"created_at"`
	UpdatedAt       string              `json:"updated_at"`
}

// newOrderResponse creates an OrderResponse from an Order model
func newOrderResponse(order *models.Order) OrderResponse {
	items := make([]OrderItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = OrderItemResponse{
			RecordID: item.RecordID,
			Title:    item.Title,
			Artist:   item.Artist,
			Price:    item.Price,
			Quantity: item.Quantity,
		}
	}

	return OrderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		Items:           items,
		Total:           order.Total,
		Status:          string(order.Status),
		ShippingAddress: order.ShippingAddress,
		CreatedAt:       order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       order.UpdatedAt.Format(time.RFC3339),
	}
}

// CreateOrder handles POST /orders
// Idempotency is handled by the idempotency middleware (configured in main.go).
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req CreateOrderRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	// Get user ID from context (set by JWT middleware)
	userID := GetUserID(r)
	if userID == "" {
		problem := zh.NewProblemDetail(http.StatusUnauthorized, "authentication required")
		return zh.Render.ProblemDetail(w, problem)
	}

	// Build order items and validate stock (atomically deduct stock)
	orderItems := make([]models.OrderItem, 0, len(req.Items))
	for _, itemReq := range req.Items {
		record, ok, err := h.store.GetRecord(ctx, itemReq.RecordID)
		if err != nil {
			return err
		}
		if !ok {
			problem := zh.NewProblemDetail(http.StatusNotFound, "record not found: "+itemReq.RecordID)
			return zh.Render.ProblemDetail(w, problem)
		}

		if record.Archived {
			problem := zh.NewProblemDetail(http.StatusConflict, "record is archived: "+itemReq.RecordID)
			return zh.Render.ProblemDetail(w, problem)
		}

		// Atomically decrement stock - prevents race conditions
		success, err := h.store.DecrementStock(ctx, itemReq.RecordID, itemReq.Quantity)
		if err != nil {
			return err
		}
		if !success {
			problem := zh.NewProblemDetail(http.StatusConflict, "insufficient stock for: "+record.Title)
			return zh.Render.ProblemDetail(w, problem)
		}

		orderItems = append(orderItems, models.OrderItem{
			RecordID: record.ID,
			Title:    record.Title,
			Artist:   record.Artist,
			Price:    record.Price,
			Quantity: itemReq.Quantity,
		})
	}

	// Create order
	order := models.NewOrder(models.OrderParams{
		UserID:          userID,
		Items:           orderItems,
		ShippingAddress: req.ShippingAddress,
	})

	// Save order
	if err := h.store.SaveOrder(ctx, order); err != nil {
		return err
	}

	return zh.RenderAndValidate(w, http.StatusCreated, newOrderResponse(order))
}

// ListOrdersRequest represents query parameters for listing orders
type ListOrdersRequest struct {
	pagination.Request
	UserID string `query:"user_id"`
	Status string `query:"status" validate:"omitempty,oneof=pending confirmed shipped delivered cancelled"`
}

// ListOrdersResponse represents a list of orders response
type ListOrdersResponse struct {
	Items []OrderResponse `json:"items"`
}

// ListOrders handles GET /orders
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req ListOrdersRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	// Get user ID from context
	userID := GetUserID(r)
	if userID == "" {
		problem := zh.NewProblemDetail(http.StatusUnauthorized, "authentication required")
		return zh.Render.ProblemDetail(w, problem)
	}

	// Get user's role
	role := GetRole(r)

	// Determine which user ID to query
	queryUserID := userID
	if role == models.RoleAdmin && req.UserID != "" {
		queryUserID = req.UserID
	}

	// Get orders with pagination from store
	orders, total, err := h.store.GetOrdersByUser(ctx, store.OrderFilter{
		BaseFilter: store.BaseFilter{
			Page:    req.Page,
			PerPage: req.PerPage,
		},
		UserID: queryUserID,
		Status: req.Status,
	})
	if err != nil {
		return err
	}

	// Apply pagination params for headers
	params := pagination.Params{
		Page:    req.Page,
		PerPage: req.PerPage,
	}.Defaults()

	// Set pagination headers
	params.WriteHeaders(w, r.URL, total)

	// Build response
	items := make([]OrderResponse, len(orders))
	for i, order := range orders {
		items[i] = newOrderResponse(order)
	}

	return zh.RenderAndValidate(w, http.StatusOK, ListOrdersResponse{
		Items: items,
	})
}

// GetOrder handles GET /orders/{id}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == "" {
		problem := zh.NewProblemDetail(http.StatusBadRequest, "order id is required")
		return zh.Render.ProblemDetail(w, problem)
	}

	order, ok, err := h.store.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		problem := zh.NewProblemDetail(http.StatusNotFound, "order not found")
		return zh.Render.ProblemDetail(w, problem)
	}

	// Check ownership or admin
	userID := GetUserID(r)
	role := GetRole(r)
	if role != models.RoleAdmin && order.UserID != userID {
		problem := zh.NewProblemDetail(http.StatusForbidden, "access denied")
		return zh.Render.ProblemDetail(w, problem)
	}

	return zh.RenderAndValidate(w, http.StatusOK, newOrderResponse(order))
}

// CancelOrder handles POST /orders/{id}/cancel
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == "" {
		problem := zh.NewProblemDetail(http.StatusBadRequest, "order id is required")
		return zh.Render.ProblemDetail(w, problem)
	}

	order, ok, err := h.store.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		problem := zh.NewProblemDetail(http.StatusNotFound, "order not found")
		return zh.Render.ProblemDetail(w, problem)
	}

	// Check ownership or admin
	userID := GetUserID(r)
	role := GetRole(r)
	if role != models.RoleAdmin && order.UserID != userID {
		problem := zh.NewProblemDetail(http.StatusForbidden, "access denied")
		return zh.Render.ProblemDetail(w, problem)
	}

	// Can only cancel pending orders
	if order.Status != models.OrderStatusPending {
		problem := zh.NewProblemDetail(http.StatusConflict, "cannot cancel order with status: "+string(order.Status))
		return zh.Render.ProblemDetail(w, problem)
	}

	// Restore stock
	for _, item := range order.Items {
		record, ok, err := h.store.GetRecord(ctx, item.RecordID)
		if err != nil {
			return err
		}
		if ok {
			record.Stock += item.Quantity
			record.UpdatedAt = time.Now()
			if err := h.store.SaveRecord(ctx, record); err != nil {
				return err
			}
		}
	}

	// Update order status
	order.Status = models.OrderStatusCancelled
	order.UpdatedAt = time.Now()
	if err := h.store.SaveOrder(ctx, order); err != nil {
		return err
	}

	return zh.R.NoContent(w)
}
