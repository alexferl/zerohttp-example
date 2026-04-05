package models

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestNewOrder_Success(t *testing.T) {
	order := NewOrder(OrderParams{
		UserID: "user-123",
		Items: []OrderItem{
			{RecordID: "rec-1", Title: "Record 1", Artist: "Artist 1", Price: 25.00, Quantity: 2},
			{RecordID: "rec-2", Title: "Record 2", Artist: "Artist 2", Price: 30.00, Quantity: 1},
		},
		ShippingAddress: Address{
			Street:  "123 Test St",
			City:    "Test City",
			State:   "Test State",
			ZipCode: "12345",
			Country: "Test Country",
		},
	})

	zhtest.AssertNotNil(t, order)
	zhtest.AssertEqual(t, "user-123", order.UserID)
	zhtest.AssertEqual(t, 80.00, order.Total) // (25*2) + (30*1)
	zhtest.AssertEqual(t, OrderStatusPending, order.Status)
	zhtest.AssertEqual(t, 2, len(order.Items))
	zhtest.AssertEqual(t, "123 Test St", order.ShippingAddress.Street)
	zhtest.AssertNotEmpty(t, order.ID)
	zhtest.AssertFalse(t, order.CreatedAt.IsZero())
	zhtest.AssertFalse(t, order.UpdatedAt.IsZero())
}

func TestNewOrder_EmptyItems(t *testing.T) {
	order := NewOrder(OrderParams{
		UserID: "user-123",
		Items:  []OrderItem{},
	})

	zhtest.AssertEqual(t, 0.00, order.Total)
	zhtest.AssertEqual(t, 0, len(order.Items))
}

func TestNewOrder_SingleItem(t *testing.T) {
	order := NewOrder(OrderParams{
		UserID: "user-456",
		Items: []OrderItem{
			{RecordID: "rec-3", Title: "Single Record", Artist: "Artist", Price: 50.00, Quantity: 1},
		},
	})

	zhtest.AssertEqual(t, 50.00, order.Total)
	zhtest.AssertEqual(t, 1, len(order.Items))
	zhtest.AssertEqual(t, "Single Record", order.Items[0].Title)
}

func TestNewOrder_QuantityGreaterThanOne(t *testing.T) {
	order := NewOrder(OrderParams{
		UserID: "user-789",
		Items: []OrderItem{
			{RecordID: "rec-4", Title: "Bulk Record", Artist: "Artist", Price: 10.00, Quantity: 5},
		},
	})

	zhtest.AssertEqual(t, 50.00, order.Total) // 10 * 5
}

func TestNewOrder_GeneratesUniqueIDs(t *testing.T) {
	order1 := NewOrder(OrderParams{
		UserID: "user-1",
		Items:  []OrderItem{{RecordID: "rec-1", Title: "Record", Artist: "Artist", Price: 20.00, Quantity: 1}},
	})
	order2 := NewOrder(OrderParams{
		UserID: "user-2",
		Items:  []OrderItem{{RecordID: "rec-2", Title: "Record", Artist: "Artist", Price: 20.00, Quantity: 1}},
	})

	zhtest.AssertNotEqual(t, order1.ID, order2.ID)
}

func TestNewOrder_SetsTimestamps(t *testing.T) {
	order := NewOrder(OrderParams{
		UserID: "user-123",
		Items:  []OrderItem{{RecordID: "rec-1", Title: "Record", Artist: "Artist", Price: 20.00, Quantity: 1}},
	})

	zhtest.AssertFalse(t, order.CreatedAt.IsZero())
	zhtest.AssertFalse(t, order.UpdatedAt.IsZero())
}

func TestNewOrder_AllOrderStatuses(t *testing.T) {
	statuses := []OrderStatus{
		OrderStatusPending,
		OrderStatusConfirmed,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusCancelled,
	}

	// Verify all status constants exist and have correct values
	expectedValues := map[OrderStatus]string{
		OrderStatusPending:   "pending",
		OrderStatusConfirmed: "confirmed",
		OrderStatusShipped:   "shipped",
		OrderStatusDelivered: "delivered",
		OrderStatusCancelled: "cancelled",
	}

	for _, status := range statuses {
		expectedValue, ok := expectedValues[status]
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, expectedValue, string(status))
	}
}

func TestOrderItem_Fields(t *testing.T) {
	item := OrderItem{
		RecordID: "rec-123",
		Title:    "Test Record",
		Artist:   "Test Artist",
		Price:    29.99,
		Quantity: 3,
	}

	zhtest.AssertEqual(t, "rec-123", item.RecordID)
	zhtest.AssertEqual(t, "Test Record", item.Title)
	zhtest.AssertEqual(t, "Test Artist", item.Artist)
	zhtest.AssertEqual(t, 29.99, item.Price)
	zhtest.AssertEqual(t, 3, item.Quantity)
}

func TestAddress_Fields(t *testing.T) {
	addr := Address{
		Street:  "456 Main St",
		City:    "New York",
		State:   "NY",
		ZipCode: "10001",
		Country: "USA",
	}

	zhtest.AssertEqual(t, "456 Main St", addr.Street)
	zhtest.AssertEqual(t, "New York", addr.City)
	zhtest.AssertEqual(t, "NY", addr.State)
	zhtest.AssertEqual(t, "10001", addr.ZipCode)
	zhtest.AssertEqual(t, "USA", addr.Country)
}
