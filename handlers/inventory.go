package handlers

import (
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/pagination"
)

// InventoryItem represents an inventory record with stock level info
type InventoryItem struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Artist        string `json:"artist"`
	Stock         int    `json:"stock"`
	LowStock      bool   `json:"low_stock"`
	ReorderNeeded bool   `json:"reorder_needed"`
}

// ListInventoryRequest represents query parameters for listing inventory
type ListInventoryRequest struct {
	pagination.Request
}

// ListInventoryResponse represents the inventory list response
type ListInventoryResponse struct {
	Items      []InventoryItem `json:"items"`
	LowStock   int             `json:"low_stock_count"`
	OutOfStock int             `json:"out_of_stock_count"`
}

// LowStockThreshold defines the stock level below which an item is considered low stock
const LowStockThreshold = 5

// ListInventory handles GET /inventory
func (h *Handler) ListInventory(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req ListInventoryRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	records, err := h.store.GetAllRecords(ctx)
	if err != nil {
		return err
	}

	allItems := make([]InventoryItem, 0, len(records))
	lowStockCount := 0
	outOfStockCount := 0

	for _, record := range records {
		item := InventoryItem{
			ID:            record.ID,
			Title:         record.Title,
			Artist:        record.Artist,
			Stock:         record.Stock,
			LowStock:      record.Stock > 0 && record.Stock <= LowStockThreshold,
			ReorderNeeded: record.Stock == 0,
		}

		allItems = append(allItems, item)

		if record.Stock == 0 {
			outOfStockCount++
		} else if record.Stock <= LowStockThreshold {
			lowStockCount++
		}
	}

	total := len(allItems)

	params := pagination.Params{
		Page:    req.Page,
		PerPage: req.PerPage,
	}.Defaults()

	offset := params.Offset()
	if offset < total {
		end := offset + params.PerPage
		if end > total {
			end = total
		}
		allItems = allItems[offset:end]
	} else {
		allItems = []InventoryItem{}
	}

	params.WriteHeaders(w, r.URL, total)

	return zh.RenderAndValidate(w, http.StatusOK, ListInventoryResponse{
		Items:      allItems,
		LowStock:   lowStockCount,
		OutOfStock: outOfStockCount,
	})
}

// RestockRequest represents a restock request
type RestockRequest struct {
	Quantity int `json:"quantity" validate:"required,min=1"`
}

// RestockResponse represents the restock operation response
type RestockResponse struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	PreviousStock int    `json:"previous_stock"`
	NewStock      int    `json:"new_stock"`
	RestockedAt   string `json:"restocked_at"`
}

// RestockRecord handles POST /inventory/{record_id}/restock
func (h *Handler) RestockRecord(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("record_id")
	if id == "" {
		problem := zh.NewProblemDetail(http.StatusBadRequest, "record id is required")
		return zh.Render.ProblemDetail(w, problem)
	}

	record, ok, err := h.store.GetRecord(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		problem := zh.NewProblemDetail(http.StatusNotFound, "record not found")
		return zh.Render.ProblemDetail(w, problem)
	}

	if record.Archived {
		problem := zh.NewProblemDetail(http.StatusConflict, "cannot restock archived record")
		return zh.Render.ProblemDetail(w, problem)
	}

	var req RestockRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	previousStock := record.Stock
	record.Stock += req.Quantity
	record.UpdatedAt = time.Now()
	if err := h.store.SaveRecord(ctx, record); err != nil {
		return err
	}

	return zh.RenderAndValidate(w, http.StatusOK, RestockResponse{
		ID:            record.ID,
		Title:         record.Title,
		PreviousStock: previousStock,
		NewStock:      record.Stock,
		RestockedAt:   time.Now().Format(time.RFC3339),
	})
}
