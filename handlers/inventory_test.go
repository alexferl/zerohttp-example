package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/stretchr/testify/mock"

	"github.com/alexferl/zerohttp-example/models"
)

func TestListInventory_Success(t *testing.T) {
	h, mockStore := setupHandler(t)

	// Create records with different stock levels
	record1 := models.NewRecord(models.RecordParams{
		Title:     "In Stock Record",
		Artist:    "Artist 1",
		Format:    models.FormatLP,
		Genre:     models.GenreJazz,
		Condition: models.ConditionNM,
		Price:     20.00,
		Stock:     10,
	})
	record2 := models.NewRecord(models.RecordParams{
		Title:     "Low Stock Record",
		Artist:    "Artist 2",
		Format:    models.FormatLP,
		Genre:     models.GenreRock,
		Condition: models.ConditionVGPlus,
		Price:     25.00,
		Stock:     3,
	})
	record3 := models.NewRecord(models.RecordParams{
		Title:     "Out of Stock Record",
		Artist:    "Artist 3",
		Format:    models.FormatLP,
		Genre:     models.GenreElectronic,
		Condition: models.ConditionMint,
		Price:     30.00,
		Stock:     0,
	})

	mockStore.On("GetAllRecords", mock.Anything).Return([]*models.Record{record1, record2, record3}, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory", nil)
	w := httptest.NewRecorder()

	err := h.ListInventory(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListInventoryResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, "3", w.Header().Get(httpx.HeaderXTotal))
	zhtest.AssertEqual(t, 1, resp.LowStock)
	zhtest.AssertEqual(t, 1, resp.OutOfStock)
	zhtest.AssertLen(t, resp.Items, 3)

	// Check the low stock item
	for _, item := range resp.Items {
		if item.ID == record2.ID {
			zhtest.AssertTrue(t, item.LowStock)
			zhtest.AssertFalse(t, item.ReorderNeeded)
		}
		if item.ID == record3.ID {
			zhtest.AssertFalse(t, item.LowStock)
			zhtest.AssertTrue(t, item.ReorderNeeded)
		}
		if item.ID == record1.ID {
			zhtest.AssertFalse(t, item.LowStock)
			zhtest.AssertFalse(t, item.ReorderNeeded)
		}
	}
}

func TestListInventory_Empty(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetAllRecords", mock.Anything).Return([]*models.Record{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/inventory", nil)
	w := httptest.NewRecorder()

	err := h.ListInventory(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp ListInventoryResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, "0", w.Header().Get(httpx.HeaderXTotal))
	zhtest.AssertEqual(t, 0, resp.LowStock)
	zhtest.AssertEqual(t, 0, resp.OutOfStock)
	zhtest.AssertLen(t, resp.Items, 0)
}

func TestRestockRecord_Success(t *testing.T) {
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
	initialStock := record.Stock

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)
	mockStore.On("SaveRecord", mock.Anything, mock.AnythingOfType("*models.Record")).Return(nil)

	req := zhtest.NewRequest(http.MethodPost, "/inventory/"+record.ID+"/restock").
		WithJSON(map[string]any{
			"quantity": 10,
		}).
		Build()
	req.SetPathValue("record_id", record.ID)
	w := httptest.NewRecorder()

	err := h.RestockRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp RestockResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, record.ID, resp.ID)
	zhtest.AssertEqual(t, record.Title, resp.Title)
	zhtest.AssertEqual(t, initialStock, resp.PreviousStock)
	zhtest.AssertEqual(t, initialStock+10, resp.NewStock)
	zhtest.AssertNotEmpty(t, resp.RestockedAt)
}

func TestRestockRecord_NotFound(t *testing.T) {
	h, mockStore := setupHandler(t)

	mockStore.On("GetRecord", mock.Anything, "nonexistent-id").Return(nil, false, nil)

	req := zhtest.NewRequest(http.MethodPost, "/inventory/nonexistent-id/restock").
		WithJSON(map[string]any{
			"quantity": 10,
		}).
		Build()
	req.SetPathValue("record_id", "nonexistent-id")
	w := httptest.NewRecorder()

	err := h.RestockRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusNotFound)
}

func TestRestockRecord_MissingID(t *testing.T) {
	h, _ := setupHandler(t)

	req := zhtest.NewRequest(http.MethodPost, "/inventory//restock").
		WithJSON(map[string]any{
			"quantity": 10,
		}).
		Build()
	req.SetPathValue("record_id", "")
	w := httptest.NewRecorder()

	err := h.RestockRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusBadRequest)
}

func TestRestockRecord_Archived(t *testing.T) {
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
	record.Archived = true

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)

	req := zhtest.NewRequest(http.MethodPost, "/inventory/"+record.ID+"/restock").
		WithJSON(map[string]any{
			"quantity": 10,
		}).
		Build()
	req.SetPathValue("record_id", record.ID)
	w := httptest.NewRecorder()

	err := h.RestockRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).ProblemDetailStatus(http.StatusConflict)
}

func TestRestockRecord_Validation_MissingQuantity(t *testing.T) {
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

	req := zhtest.NewRequest(http.MethodPost, "/inventory/"+record.ID+"/restock").
		WithJSON(map[string]any{}).
		Build()
	req.SetPathValue("record_id", record.ID)
	w := httptest.NewRecorder()

	err := h.RestockRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestRestockRecord_Validation_InvalidQuantity(t *testing.T) {
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

	req := zhtest.NewRequest(http.MethodPost, "/inventory/"+record.ID+"/restock").
		WithJSON(map[string]any{
			"quantity": 0,
		}).
		Build()
	req.SetPathValue("record_id", record.ID)
	w := httptest.NewRecorder()

	err := h.RestockRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestRestockRecord_Validation_NegativeQuantity(t *testing.T) {
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

	req := zhtest.NewRequest(http.MethodPost, "/inventory/"+record.ID+"/restock").
		WithJSON(map[string]any{
			"quantity": -5,
		}).
		Build()
	req.SetPathValue("record_id", record.ID)
	w := httptest.NewRecorder()

	err := h.RestockRecord(w, req)
	zhtest.AssertError(t, err)
}

func TestRestockRecord_LargeQuantity(t *testing.T) {
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
	initialStock := record.Stock

	mockStore.On("GetRecord", mock.Anything, record.ID).Return(record, true, nil)
	mockStore.On("SaveRecord", mock.Anything, mock.AnythingOfType("*models.Record")).Return(nil)

	req := zhtest.NewRequest(http.MethodPost, "/inventory/"+record.ID+"/restock").
		WithJSON(map[string]any{
			"quantity": 1000,
		}).
		Build()
	req.SetPathValue("record_id", record.ID)
	w := httptest.NewRecorder()

	err := h.RestockRecord(w, req)
	zhtest.AssertNoError(t, err)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	var resp RestockResponse
	zhtest.AssertWith(t, w).JSON(&resp)

	zhtest.AssertEqual(t, initialStock, resp.PreviousStock)
	zhtest.AssertEqual(t, initialStock+1000, resp.NewStock)
}
