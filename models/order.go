package models

import (
	"time"

	"github.com/rs/xid"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// OrderItem represents a single item in an order
type OrderItem struct {
	RecordID string  `json:"record_id" bson:"record_id"`
	Title    string  `json:"title" bson:"title"`
	Artist   string  `json:"artist" bson:"artist"`
	Price    float64 `json:"price" bson:"price"`
	Quantity int     `json:"quantity" bson:"quantity"`
}

// Order represents a customer order.
type Order struct {
	BaseModel       `bson:",inline"`
	UserID          string      `json:"user_id" bson:"user_id"`
	Items           []OrderItem `json:"items" bson:"items"`
	Total           float64     `json:"total" bson:"total"`
	Status          OrderStatus `json:"status" bson:"status"`
	ShippingAddress Address     `json:"shipping_address" bson:"shipping_address"`
}

// OrderParams contains parameters for creating a new order.
type OrderParams struct {
	UserID          string
	Items           []OrderItem
	ShippingAddress Address
}

// NewOrder creates a new order with generated ID, calculated total, and timestamps.
func NewOrder(p OrderParams) *Order {
	total := 0.0
	for _, item := range p.Items {
		total += item.Price * float64(item.Quantity)
	}

	return &Order{
		BaseModel: BaseModel{
			ID:        xid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:          p.UserID,
		Items:           p.Items,
		Total:           total,
		Status:          OrderStatusPending,
		ShippingAddress: p.ShippingAddress,
	}
}
