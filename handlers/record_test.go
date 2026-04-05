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

func TestCreateRecord_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("SaveRecord", mock.Anything, mock.AnythingOfType("*models.Record")).Return(nil)

	req := zhtest.NewRequest(http.MethodPost, "/records").
		WithJSON(map[string]any{
			"title":          "Kind of Blue",
			"artist":         "Miles Davis",
			"year":           1959,
			"label":          "Columbia",
			"catalog_number": "CL 1355",
			"format":         "lp",
			"genre":          "jazz",
			"condition":      "nm",
			"price":          29.99,
			"stock":          5,
			"description":    "Classic jazz album",
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusCreated)

	var resp RecordResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertNotEmpty(t, resp.ID)
	zhtest.AssertEqual(t, "Kind of Blue", resp.Title)
	zhtest.AssertEqual(t, "Miles Davis", resp.Artist)
	zhtest.AssertEqual(t, 1959, resp.Year)
	zhtest.AssertEqual(t, "lp", string(resp.Format))
	zhtest.AssertEqual(t, "jazz", string(resp.Genre))
	zhtest.AssertEqual(t, "nm", string(resp.Condition))
	zhtest.AssertEqual(t, 29.99, resp.Price)
	zhtest.AssertEqual(t, 5, resp.Stock)
	zhtest.AssertFalse(t, resp.Archived)
}

func TestCreateRecord_MinimalFields(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("SaveRecord", mock.Anything, mock.AnythingOfType("*models.Record")).Return(nil)

	req := zhtest.NewRequest(http.MethodPost, "/records").
		WithJSON(map[string]any{
			"title":     "Abbey Road",
			"artist":    "The Beatles",
			"year":      1969,
			"format":    "lp",
			"genre":     "rock",
			"condition": "vg+",
			"price":     25.00,
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusCreated)

	var resp RecordResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, "Abbey Road", resp.Title)
	zhtest.AssertEqual(t, 1969, resp.Year)
	zhtest.AssertEqual(t, 0, resp.Stock)
}

func TestCreateRecord_Validation_MissingTitle(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/records").
		WithJSON(map[string]any{
			"artist":    "The Beatles",
			"format":    "lp",
			"genre":     "rock",
			"condition": "vg+",
			"price":     25.00,
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateRecord_Validation_InvalidGenre(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/records").
		WithJSON(map[string]any{
			"title":     "Test Album",
			"artist":    "Test Artist",
			"format":    "lp",
			"genre":     "invalid-genre",
			"condition": "nm",
			"price":     25.00,
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateRecord_Validation_InvalidCondition(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/records").
		WithJSON(map[string]any{
			"title":     "Test Album",
			"artist":    "Test Artist",
			"format":    "lp",
			"genre":     "rock",
			"condition": "poor",
			"price":     25.00,
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateRecord_Validation_InvalidFormat(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/records").
		WithJSON(map[string]any{
			"title":     "Test Album",
			"artist":    "Test Artist",
			"format":    "cd",
			"genre":     "rock",
			"condition": "nm",
			"price":     25.00,
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateRecord_Validation_NegativePrice(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/records").
		WithJSON(map[string]any{
			"title":     "Test Album",
			"artist":    "Test Artist",
			"format":    "lp",
			"genre":     "rock",
			"condition": "nm",
			"price":     -10.00,
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestCreateRecord_Validation_InvalidYear(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/records").
		WithJSON(map[string]any{
			"title":     "Test Album",
			"artist":    "Test Artist",
			"year":      1800,
			"format":    "lp",
			"genre":     "rock",
			"condition": "nm",
			"price":     25.00,
		}).
		Build()
	w := httptest.NewRecorder()

	err := h.CreateRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestGetRecord_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	record := models.NewRecord(models.RecordParams{
		Title:         "Test Record",
		Artist:        "Test Artist",
		Year:          2020,
		Label:         "Test Label",
		CatalogNumber: "TEST-001",
		Format:        models.FormatLP,
		Genre:         models.GenreRock,
		Condition:     models.ConditionNM,
		Price:         19.99,
		Stock:         10,
		Description:   "A test record",
	})

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)

	req := httptest.NewRequest(http.MethodGet, "/records/"+record.ID, nil)
	req.SetPathValue("id", record.ID)
	w := httptest.NewRecorder()

	err := h.GetRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp RecordResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, record.ID, resp.ID)
	zhtest.AssertEqual(t, record.Title, resp.Title)
	zhtest.AssertEqual(t, record.Artist, resp.Artist)
}

func TestGetRecord_NotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetRecord", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := httptest.NewRequest(http.MethodGet, "/records/nonexistent-id", nil)
	req.SetPathValue("id", "nonexistent-id")
	w := httptest.NewRecorder()

	err := h.GetRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestGetRecord_MissingID(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/records/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	err := h.GetRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusBadRequest)
}

func TestGetRecord_Archived(t *testing.T) {
	h, mockStore := setupHandler(t)

	record := models.NewRecord(models.RecordParams{
		Title:     "Test Record",
		Artist:    "Test Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreRock,
		Condition: models.ConditionNM,
		Price:     19.99,
		Stock:     10,
	})
	record.Archived = true

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)

	req := httptest.NewRequest(http.MethodGet, "/records/"+record.ID, nil)
	req.SetPathValue("id", record.ID)
	w := httptest.NewRecorder()

	err := h.GetRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestListRecords_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	// Create multiple records
	records := []*models.Record{
		models.NewRecord(models.RecordParams{
			Title:     "Record A",
			Artist:    "Artist 1",
			Format:    models.FormatLP,
			Genre:     models.GenreJazz,
			Condition: models.ConditionNM,
			Price:     20.00,
			Stock:     5,
		}),
		models.NewRecord(models.RecordParams{
			Title:     "Record B",
			Artist:    "Artist 2",
			Format:    models.FormatLP,
			Genre:     models.GenreJazz,
			Condition: models.ConditionNM,
			Price:     25.00,
			Stock:     5,
		}),
		models.NewRecord(models.RecordParams{
			Title:     "Record C",
			Artist:    "Artist 3",
			Format:    models.FormatLP,
			Genre:     models.GenreJazz,
			Condition: models.ConditionNM,
			Price:     30.00,
			Stock:     5,
		}),
	}

	mockStore.On("GetFilteredRecords", mock.Anything, store.RecordFilter{BaseFilter: store.BaseFilter{Page: 1, PerPage: 25}}).Return(records, 3, nil)

	req := httptest.NewRequest(http.MethodGet, "/records", nil)
	w := httptest.NewRecorder()

	err := h.ListRecords(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListRecordsResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertLen(t, resp.Items, 3)
	zhtest.AssertEqual(t, "3", w.Header().Get(httpx.HeaderXTotal))
}

func TestListRecords_WithFilters(t *testing.T) {
	h, mockStore := setupHandler(t)

	jazzRecord := models.NewRecord(models.RecordParams{
		Title:     "Jazz Album",
		Artist:    "Jazz Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreJazz,
		Condition: models.ConditionVGPlus,
		Price:     30.00,
		Stock:     3,
	})

	mockStore.On("GetFilteredRecords", mock.Anything, store.RecordFilter{BaseFilter: store.BaseFilter{Page: 1, PerPage: 25}, Genres: []string{"jazz"}}).Return([]*models.Record{jazzRecord}, 1, nil)

	// Test filtering by genre
	req := httptest.NewRequest(http.MethodGet, "/records?genre=jazz", nil)
	w := httptest.NewRecorder()

	err := h.ListRecords(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListRecordsResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	// Should only return jazz records
	zhtest.AssertEqual(t, "1", w.Header().Get(httpx.HeaderXTotal))
	zhtest.AssertEqual(t, "Jazz Album", resp.Items[0].Title)
}

func TestListRecords_WithLimit(t *testing.T) {
	h, mockStore := setupHandler(t)

	// Create multiple records
	records := []*models.Record{
		models.NewRecord(models.RecordParams{
			Title: "Record A", Artist: "Artist", Format: models.FormatLP,
			Genre: models.GenreJazz, Condition: models.ConditionNM, Price: 20.00, Stock: 5,
		}),
		models.NewRecord(models.RecordParams{
			Title: "Record B", Artist: "Artist", Format: models.FormatLP,
			Genre: models.GenreJazz, Condition: models.ConditionNM, Price: 20.00, Stock: 5,
		}),
	}

	mockStore.On("GetFilteredRecords", mock.Anything, store.RecordFilter{BaseFilter: store.BaseFilter{Page: 1, PerPage: 2}}).Return(records, 2, nil)

	// Test with per_page
	req := httptest.NewRequest(http.MethodGet, "/records?per_page=2", nil)
	w := httptest.NewRecorder()

	err := h.ListRecords(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListRecordsResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, 2, len(resp.Items))
}

func TestListRecords_Empty(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetFilteredRecords", mock.Anything, store.RecordFilter{BaseFilter: store.BaseFilter{Page: 1, PerPage: 25}}).Return([]*models.Record{}, 0, nil)

	req := httptest.NewRequest(http.MethodGet, "/records", nil)
	w := httptest.NewRecorder()

	err := h.ListRecords(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListRecordsResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertLen(t, resp.Items, 0)
	zhtest.AssertEqual(t, "0", w.Header().Get(httpx.HeaderXTotal))
}

func TestUpdateRecord_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	record := models.NewRecord(models.RecordParams{
		Title:         "Test Record",
		Artist:        "Test Artist",
		Year:          2020,
		Label:         "Test Label",
		CatalogNumber: "TEST-001",
		Format:        models.FormatLP,
		Genre:         models.GenreRock,
		Condition:     models.ConditionNM,
		Price:         19.99,
		Stock:         10,
		Description:   "A test record",
	})

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)
	mockStore.On("SaveRecord", mock.Anything, mock.AnythingOfType("*models.Record")).Return(nil)

	req := zhtest.NewRequest(http.MethodPatch, "/records/"+record.ID).
		WithJSON(map[string]any{
			"title": "Updated Title",
			"price": 29.99,
		}).
		Build()
	req.SetPathValue("id", record.ID)
	w := httptest.NewRecorder()

	err := h.UpdateRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp RecordResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, "Updated Title", resp.Title)
	zhtest.AssertEqual(t, 29.99, resp.Price)
	zhtest.AssertEqual(t, record.Artist, resp.Artist) // Unchanged
}

func TestUpdateRecord_NotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetRecord", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := zhtest.NewRequest(http.MethodPatch, "/records/nonexistent-id").
		WithJSON(map[string]any{
			"title": "Updated Title",
		}).
		Build()
	req.SetPathValue("id", "nonexistent-id")
	w := httptest.NewRecorder()

	err := h.UpdateRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestUpdateRecord_MissingID(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPatch, "/records/").
		WithJSON(map[string]any{
			"title": "Updated Title",
		}).
		Build()
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	err := h.UpdateRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusBadRequest)
}

func TestUpdateRecord_Validation_InvalidCondition(t *testing.T) {
	h, mockStore := setupHandler(t)

	record := models.NewRecord(models.RecordParams{
		Title:     "Test Record",
		Artist:    "Test Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreRock,
		Condition: models.ConditionNM,
		Price:     19.99,
		Stock:     10,
	})

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)

	req := zhtest.NewRequest(http.MethodPatch, "/records/"+record.ID).
		WithJSON(map[string]any{
			"condition": "invalid",
		}).
		Build()
	req.SetPathValue("id", record.ID)
	w := httptest.NewRecorder()

	err := h.UpdateRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestUpdateRecord_UpdateStockToZero(t *testing.T) {
	h, mockStore := setupHandler(t)

	record := models.NewRecord(models.RecordParams{
		Title:       "Test Record",
		Artist:      "Test Artist",
		Format:      models.FormatLP,
		Genre:       models.GenreRock,
		Condition:   models.ConditionNM,
		Price:       19.99,
		Stock:       10,
		Description: "A test record",
	})
	// Record starts with stock = 10

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)
	mockStore.On("SaveRecord", mock.Anything, mock.AnythingOfType("*models.Record")).Return(nil)

	req := zhtest.NewRequest(http.MethodPatch, "/records/"+record.ID).
		WithJSON(map[string]any{
			"stock": 0,
		}).
		Build()
	req.SetPathValue("id", record.ID)
	w := httptest.NewRecorder()

	err := h.UpdateRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp RecordResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, 0, resp.Stock)
}

func TestUpdateRecord_GetRecordError(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetRecord", mock.Anything, "record-id").Return(nil, false, assert.AnError)

	req := zhtest.NewRequest(http.MethodPatch, "/records/record-id").
		WithJSON(map[string]any{
			"title": "Updated Title",
		}).
		Build()
	req.SetPathValue("id", "record-id")
	w := httptest.NewRecorder()

	err := h.UpdateRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestUpdateRecord_SaveRecordError(t *testing.T) {
	h, mockStore := setupHandler(t)

	record := models.NewRecord(models.RecordParams{
		Title:     "Test Record",
		Artist:    "Test Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreRock,
		Condition: models.ConditionNM,
		Price:     19.99,
		Stock:     10,
	})

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)
	mockStore.On("SaveRecord", mock.Anything, mock.AnythingOfType("*models.Record")).Return(assert.AnError)

	req := zhtest.NewRequest(http.MethodPatch, "/records/"+record.ID).
		WithJSON(map[string]any{
			"title": "Updated Title",
		}).
		Build()
	req.SetPathValue("id", record.ID)
	w := httptest.NewRecorder()

	err := h.UpdateRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestArchiveRecord_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	record := models.NewRecord(models.RecordParams{
		Title:       "Test Record",
		Artist:      "Test Artist",
		Format:      models.FormatLP,
		Genre:       models.GenreRock,
		Condition:   models.ConditionNM,
		Price:       19.99,
		Stock:       10,
		Description: "A test record",
	})

	// Verify record starts not archived
	zhtest.AssertFalse(t, record.Archived)

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)
	mockStore.On("SaveRecord", mock.Anything, mock.AnythingOfType("*models.Record")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/records/"+record.ID+"/archive", nil)
	req.SetPathValue("id", record.ID)
	w := httptest.NewRecorder()

	err := h.ArchiveRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusNoContent)
	zhtest.AssertWith(t, w).BodyEmpty()
}

func TestArchiveRecord_NotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetRecord", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := httptest.NewRequest(http.MethodPost, "/records/nonexistent-id/archive", nil)
	req.SetPathValue("id", "nonexistent-id")
	w := httptest.NewRecorder()

	err := h.ArchiveRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestArchiveRecord_MissingID(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/records//archive", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	err := h.ArchiveRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusBadRequest)
}

func TestArchiveRecord_AlreadyArchived(t *testing.T) {
	h, mockStore := setupHandler(t)

	record := models.NewRecord(models.RecordParams{
		Title:       "Test Record",
		Artist:      "Test Artist",
		Format:      models.FormatLP,
		Genre:       models.GenreRock,
		Condition:   models.ConditionNM,
		Price:       19.99,
		Stock:       10,
		Description: "A test record",
	})
	record.Archived = true

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)

	req := httptest.NewRequest(http.MethodPost, "/records/"+record.ID+"/archive", nil)
	req.SetPathValue("id", record.ID)
	w := httptest.NewRecorder()

	err := h.ArchiveRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusConflict)
}
