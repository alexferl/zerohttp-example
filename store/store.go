package store

import (
	"context"

	"github.com/alexferl/zerohttp-example/models"
)

// UserStore defines the interface for user storage
type UserStore interface {
	GetUser(ctx context.Context, id string) (*models.User, bool, error)
	SaveUser(ctx context.Context, u *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, bool, error)
	GetAllUsers(ctx context.Context, filter UserFilter) ([]*models.User, int, error)
}

// BaseFilter contains common pagination fields
type BaseFilter struct {
	Page    int
	PerPage int
}

// RecordFilter contains filter options for GetFilteredRecords
type RecordFilter struct {
	BaseFilter
	Genres     []string
	Decade     int
	Conditions []string
	Formats    []string
	Artist     string
	MinPrice   float64
	MaxPrice   float64
	InStock    bool
	Sort       string
}

// RecordStore defines the interface for record storage
type RecordStore interface {
	GetRecord(ctx context.Context, id string) (*models.Record, bool, error)
	SaveRecord(ctx context.Context, r *models.Record) error
	GetAllRecords(ctx context.Context) ([]*models.Record, error)
	GetFilteredRecords(ctx context.Context, filter RecordFilter) ([]*models.Record, int, error)
	DecrementStock(ctx context.Context, id string, quantity int) (bool, error)
}

// OrderFilter contains filter options for GetOrdersByUser
type OrderFilter struct {
	BaseFilter
	UserID string
	Status string
}

// UserFilter contains filter options for GetAllUsers
type UserFilter struct {
	BaseFilter
}

// AllOrdersFilter contains filter options for GetAllOrders
type AllOrdersFilter struct {
	BaseFilter
}

// OrderStore defines the interface for order storage
type OrderStore interface {
	GetOrder(ctx context.Context, id string) (*models.Order, bool, error)
	SaveOrder(ctx context.Context, o *models.Order) error
	GetOrdersByUser(ctx context.Context, filter OrderFilter) ([]*models.Order, int, error)
	GetAllOrders(ctx context.Context, filter AllOrdersFilter) ([]*models.Order, int, error)
}

// Store combines all in-memory storage interfaces.
// Token and idempotency storage are handled by Redis separately.
type Store interface {
	UserStore
	RecordStore
	OrderStore
}
