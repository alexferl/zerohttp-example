package models

// Address represents a physical address
type Address struct {
	Street  string `json:"street" bson:"street" validate:"required,min=1,max=200"`
	City    string `json:"city" bson:"city" validate:"required,min=1,max=100"`
	State   string `json:"state" bson:"state" validate:"required,min=1,max=100"`
	ZipCode string `json:"zip_code" bson:"zip_code" validate:"required,min=1,max=20"`
	Country string `json:"country" bson:"country" validate:"required,min=1,max=100"`
}
