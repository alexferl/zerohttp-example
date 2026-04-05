package models

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestNewRecord_Success(t *testing.T) {
	record := NewRecord(RecordParams{
		Title:         "Kind of Blue",
		Artist:        "Miles Davis",
		Year:          1959,
		Label:         "Columbia",
		CatalogNumber: "CL 1355",
		Format:        FormatLP,
		Genre:         GenreJazz,
		Condition:     ConditionNM,
		Price:         29.99,
		Stock:         5,
		Description:   "Classic jazz album",
	})

	zhtest.AssertNotNil(t, record)
	zhtest.AssertEqual(t, "Kind of Blue", record.Title)
	zhtest.AssertEqual(t, "Miles Davis", record.Artist)
	zhtest.AssertEqual(t, 1959, record.Year)
	zhtest.AssertEqual(t, "Columbia", record.Label)
	zhtest.AssertEqual(t, "CL 1355", record.CatalogNumber)
	zhtest.AssertEqual(t, FormatLP, record.Format)
	zhtest.AssertEqual(t, GenreJazz, record.Genre)
	zhtest.AssertEqual(t, ConditionNM, record.Condition)
	zhtest.AssertEqual(t, 29.99, record.Price)
	zhtest.AssertEqual(t, 5, record.Stock)
	zhtest.AssertEqual(t, "Classic jazz album", record.Description)
	zhtest.AssertFalse(t, record.Archived)
	zhtest.AssertNotEmpty(t, record.ID)
	zhtest.AssertFalse(t, record.CreatedAt.IsZero())
	zhtest.AssertFalse(t, record.UpdatedAt.IsZero())
}

func TestNewRecord_MinimalParams(t *testing.T) {
	record := NewRecord(RecordParams{
		Title:     "Test Album",
		Artist:    "Test Artist",
		Format:    FormatLP,
		Genre:     GenreRock,
		Condition: ConditionVGPlus,
		Price:     25.00,
	})

	zhtest.AssertEqual(t, "Test Album", record.Title)
	zhtest.AssertEqual(t, 0, record.Year)
	zhtest.AssertEqual(t, "", record.Label)
	zhtest.AssertEqual(t, 0, record.Stock)
	zhtest.AssertEqual(t, "", record.Description)
}

func TestNewRecord_GeneratesUniqueIDs(t *testing.T) {
	record1 := NewRecord(RecordParams{
		Title:     "Record 1",
		Artist:    "Artist 1",
		Format:    FormatLP,
		Genre:     GenreJazz,
		Condition: ConditionNM,
		Price:     20.00,
	})
	record2 := NewRecord(RecordParams{
		Title:     "Record 2",
		Artist:    "Artist 2",
		Format:    FormatLP,
		Genre:     GenreRock,
		Condition: ConditionNM,
		Price:     25.00,
	})

	zhtest.AssertNotEqual(t, record1.ID, record2.ID)
}

func TestNewRecord_SetsTimestamps(t *testing.T) {
	record := NewRecord(RecordParams{
		Title:     "Test Album",
		Artist:    "Test Artist",
		Format:    FormatLP,
		Genre:     GenreJazz,
		Condition: ConditionNM,
		Price:     20.00,
	})

	zhtest.AssertFalse(t, record.CreatedAt.IsZero())
	zhtest.AssertFalse(t, record.UpdatedAt.IsZero())
}

func TestNewRecord_AllGenres(t *testing.T) {
	genres := []Genre{
		GenreJazz, GenreRock, GenreElectronic, GenreHipHop,
		GenreClassical, GenreSoul, GenreFunk, GenreBlues,
	}

	for _, genre := range genres {
		record := NewRecord(RecordParams{
			Title:     "Test",
			Artist:    "Test",
			Format:    FormatLP,
			Genre:     genre,
			Condition: ConditionNM,
			Price:     20.00,
		})
		zhtest.AssertEqual(t, genre, record.Genre)
	}
}

func TestNewRecord_AllFormats(t *testing.T) {
	formats := []Format{FormatLP, Format7In, Format10In, Format12In}

	for _, format := range formats {
		record := NewRecord(RecordParams{
			Title:     "Test",
			Artist:    "Test",
			Format:    format,
			Genre:     GenreJazz,
			Condition: ConditionNM,
			Price:     20.00,
		})
		zhtest.AssertEqual(t, format, record.Format)
	}
}

func TestNewRecord_AllConditions(t *testing.T) {
	conditions := []Condition{
		ConditionMint, ConditionNMMinus, ConditionNM,
		ConditionVGPlus, ConditionVG, ConditionGPlus,
	}

	for _, condition := range conditions {
		record := NewRecord(RecordParams{
			Title:     "Test",
			Artist:    "Test",
			Format:    FormatLP,
			Genre:     GenreJazz,
			Condition: condition,
			Price:     20.00,
		})
		zhtest.AssertEqual(t, condition, record.Condition)
	}
}
