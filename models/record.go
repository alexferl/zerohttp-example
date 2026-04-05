package models

import (
	"time"

	"github.com/rs/xid"
)

// Format represents vinyl record formats
type Format string

const (
	FormatLP   Format = "lp"
	Format7In  Format = "7in"
	Format10In Format = "10in"
	Format12In Format = "12in"
)

// Genre represents music genres
type Genre string

const (
	GenreJazz       Genre = "jazz"
	GenreRock       Genre = "rock"
	GenreElectronic Genre = "electronic"
	GenreHipHop     Genre = "hip-hop"
	GenreClassical  Genre = "classical"
	GenreSoul       Genre = "soul"
	GenreFunk       Genre = "funk"
	GenreBlues      Genre = "blues"
)

// Condition represents record condition grading
type Condition string

const (
	ConditionMint    Condition = "mint"
	ConditionNMMinus Condition = "nm-"
	ConditionNM      Condition = "nm"
	ConditionVGPlus  Condition = "vg+"
	ConditionVG      Condition = "vg"
	ConditionGPlus   Condition = "g+"
)

// Record represents a vinyl record in the catalog.
type Record struct {
	BaseModel     `bson:",inline"`
	Title         string    `json:"title" bson:"title" validate:"required,min=1,max=200"`
	Artist        string    `json:"artist" bson:"artist" validate:"required,min=1,max=100"`
	Year          int       `json:"year" bson:"year" validate:"min=1900,max=2100"`
	Label         string    `json:"label" bson:"label"`
	CatalogNumber string    `json:"catalog_number" bson:"catalog_number"`
	Format        Format    `json:"format" bson:"format" validate:"required,oneof=lp 7in 10in 12in"`
	Genre         Genre     `json:"genre" bson:"genre" validate:"required,oneof=jazz rock electronic hip-hop classical soul funk blues"`
	Condition     Condition `json:"condition" bson:"condition" validate:"required,oneof=mint nm- nm vg+ vg g+"`
	Price         float64   `json:"price" bson:"price" validate:"required,min=0"`
	Stock         int       `json:"stock" bson:"stock" validate:"min=0"`
	Description   string    `json:"description" bson:"description"`
	Archived      bool      `json:"archived" bson:"archived"`
}

// RecordParams contains parameters for creating a new record.
type RecordParams struct {
	Title         string
	Artist        string
	Year          int
	Label         string
	CatalogNumber string
	Format        Format
	Genre         Genre
	Condition     Condition
	Price         float64
	Stock         int
	Description   string
}

// NewRecord creates a new record with generated ID and timestamps.
func NewRecord(p RecordParams) *Record {
	return &Record{
		BaseModel: BaseModel{
			ID:        xid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Title:         p.Title,
		Artist:        p.Artist,
		Year:          p.Year,
		Label:         p.Label,
		CatalogNumber: p.CatalogNumber,
		Format:        p.Format,
		Genre:         p.Genre,
		Condition:     p.Condition,
		Price:         p.Price,
		Stock:         p.Stock,
		Description:   p.Description,
		Archived:      false,
	}
}
