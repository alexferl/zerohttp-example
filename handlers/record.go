package handlers

import (
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/pagination"

	"github.com/alexferl/zerohttp-example/models"
	"github.com/alexferl/zerohttp-example/store"
)

// CreateRecordRequest represents a record creation request
type CreateRecordRequest struct {
	Title         string           `json:"title" validate:"required,min=1,max=200"`
	Artist        string           `json:"artist" validate:"required,min=1,max=100"`
	Year          int              `json:"year" validate:"min=1900,max=2100"`
	Label         string           `json:"label"`
	CatalogNumber string           `json:"catalog_number"`
	Format        models.Format    `json:"format" validate:"required,oneof=lp 7in 10in 12in"`
	Genre         models.Genre     `json:"genre" validate:"required,oneof=jazz rock electronic hip-hop classical soul funk blues"`
	Condition     models.Condition `json:"condition" validate:"required,oneof=mint nm- nm vg+ vg g+"`
	Price         float64          `json:"price" validate:"required,min=0"`
	Stock         int              `json:"stock" validate:"min=0"`
	Description   string           `json:"description"`
}

// RecordResponse represents a record response
type RecordResponse struct {
	ID            string           `json:"id"`
	Title         string           `json:"title"`
	Artist        string           `json:"artist"`
	Year          int              `json:"year"`
	Label         string           `json:"label"`
	CatalogNumber string           `json:"catalog_number"`
	Format        models.Format    `json:"format"`
	Genre         models.Genre     `json:"genre"`
	Condition     models.Condition `json:"condition"`
	Price         float64          `json:"price"`
	Stock         int              `json:"stock"`
	Description   string           `json:"description"`
	Archived      bool             `json:"archived"`
	CreatedAt     string           `json:"created_at"`
	UpdatedAt     string           `json:"updated_at"`
}

// newRecordResponse creates a RecordResponse from a Record model
func newRecordResponse(record *models.Record) RecordResponse {
	return RecordResponse{
		ID:            record.ID,
		Title:         record.Title,
		Artist:        record.Artist,
		Year:          record.Year,
		Label:         record.Label,
		CatalogNumber: record.CatalogNumber,
		Format:        record.Format,
		Genre:         record.Genre,
		Condition:     record.Condition,
		Price:         record.Price,
		Stock:         record.Stock,
		Description:   record.Description,
		Archived:      record.Archived,
		CreatedAt:     record.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     record.UpdatedAt.Format(time.RFC3339),
	}
}

// CreateRecord handles POST /records
func (h *Handler) CreateRecord(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req CreateRecordRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	record := models.NewRecord(models.RecordParams{
		Title:         req.Title,
		Artist:        req.Artist,
		Year:          req.Year,
		Label:         req.Label,
		CatalogNumber: req.CatalogNumber,
		Format:        req.Format,
		Genre:         req.Genre,
		Condition:     req.Condition,
		Price:         req.Price,
		Stock:         req.Stock,
		Description:   req.Description,
	})

	if err := h.store.SaveRecord(ctx, record); err != nil {
		return err
	}

	return zh.RenderAndValidate(w, http.StatusCreated, newRecordResponse(record))
}

// ListRecordsRequest represents query parameters for listing records
type ListRecordsRequest struct {
	pagination.Request
	Genre     []string `query:"genre" validate:"omitempty,anyof=jazz rock electronic hip-hop classical soul funk blues"`
	Decade    int      `query:"decade" validate:"omitempty,min=1900,max=2100"`
	Condition []string `query:"condition" validate:"omitempty,anyof=mint nm- nm vg+ vg g+"`
	Format    []string `query:"format" validate:"omitempty,anyof=lp 7in 10in 12in"`
	Artist    string   `query:"artist"`
	MinPrice  float64  `query:"min_price" validate:"omitempty,min=0"`
	MaxPrice  float64  `query:"max_price" validate:"omitempty,min=0"`
	InStock   bool     `query:"in_stock"`
	Sort      string   `query:"sort" validate:"omitempty,oneof=price_asc price_desc year_asc year_desc created_desc"`
}

// ListRecordsResponse represents a list of records response
type ListRecordsResponse struct {
	Items []RecordResponse `json:"items"`
}

// ListRecords handles GET /records
func (h *Handler) ListRecords(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req ListRecordsRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	params := pagination.Params{
		Page:    req.Page,
		PerPage: req.PerPage,
	}.Defaults()

	records, total, err := h.store.GetFilteredRecords(ctx, store.RecordFilter{
		BaseFilter: store.BaseFilter{
			Page:    params.Page,
			PerPage: params.PerPage,
		},
		Genres:     req.Genre,
		Decade:     req.Decade,
		Conditions: req.Condition,
		Formats:    req.Format,
		Artist:     req.Artist,
		MinPrice:   req.MinPrice,
		MaxPrice:   req.MaxPrice,
		InStock:    req.InStock,
		Sort:       req.Sort,
	})
	if err != nil {
		return err
	}

	items := make([]RecordResponse, len(records))
	for i, record := range records {
		items[i] = newRecordResponse(record)
	}

	// Set pagination headers
	params.WriteHeaders(w, r.URL, total)

	response := ListRecordsResponse{
		Items: items,
	}

	return zh.RenderAndValidate(w, http.StatusOK, response)
}

// GetRecord handles GET /records/{id}
func (h *Handler) GetRecord(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("id")
	if id == "" {
		problem := zh.NewProblemDetail(http.StatusBadRequest, "record id is required")
		return zh.Render.ProblemDetail(w, problem)
	}

	record, ok, err := h.store.GetRecord(ctx, id)
	if err != nil {
		return err
	}
	if !ok || record.Archived {
		problem := zh.NewProblemDetail(http.StatusNotFound, "record not found")
		return zh.Render.ProblemDetail(w, problem)
	}

	return zh.RenderAndValidate(w, http.StatusOK, newRecordResponse(record))
}

// UpdateRecordRequest represents a record update request
type UpdateRecordRequest struct {
	Title         string           `json:"title" validate:"omitempty,min=1,max=200"`
	Artist        string           `json:"artist" validate:"omitempty,min=1,max=100"`
	Year          int              `json:"year" validate:"omitempty,min=1900,max=2100"`
	Label         string           `json:"label"`
	CatalogNumber string           `json:"catalog_number"`
	Format        models.Format    `json:"format" validate:"omitempty,oneof=lp 7in 10in 12in"`
	Genre         models.Genre     `json:"genre" validate:"omitempty,oneof=jazz rock electronic hip-hop classical soul funk blues"`
	Condition     models.Condition `json:"condition" validate:"omitempty,oneof=mint nm- nm vg+ vg g+"`
	Price         float64          `json:"price" validate:"omitempty,min=0"`
	Stock         int              `json:"stock" validate:"omitempty,min=0"`
	Description   string           `json:"description"`
}

// UpdateRecord handles PATCH /records/{id}
func (h *Handler) UpdateRecord(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("id")
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

	var req UpdateRecordRequest
	if err := zh.BindAndValidate(r, &req); err != nil {
		return err
	}

	if req.Title != "" {
		record.Title = req.Title
	}
	if req.Artist != "" {
		record.Artist = req.Artist
	}
	if req.Year != 0 {
		record.Year = req.Year
	}
	if req.Label != "" {
		record.Label = req.Label
	}
	if req.CatalogNumber != "" {
		record.CatalogNumber = req.CatalogNumber
	}
	if req.Format != "" {
		record.Format = req.Format
	}
	if req.Genre != "" {
		record.Genre = req.Genre
	}
	if req.Condition != "" {
		record.Condition = req.Condition
	}
	if req.Price != 0 {
		record.Price = req.Price
	}
	if req.Stock != 0 || (req.Stock == 0 && r.Body != http.NoBody) {
		record.Stock = req.Stock
	}
	if req.Description != "" {
		record.Description = req.Description
	}

	record.UpdatedAt = time.Now()
	if err := h.store.SaveRecord(ctx, record); err != nil {
		return err
	}

	return zh.RenderAndValidate(w, http.StatusOK, newRecordResponse(record))
}

// ArchiveRecord handles POST /records/{id}/archive
func (h *Handler) ArchiveRecord(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := r.PathValue("id")
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
		problem := zh.NewProblemDetail(http.StatusConflict, "record is already archived")
		return zh.Render.ProblemDetail(w, problem)
	}

	record.Archived = true
	record.UpdatedAt = time.Now()
	if err := h.store.SaveRecord(ctx, record); err != nil {
		return err
	}

	return zh.R.NoContent(w)
}
