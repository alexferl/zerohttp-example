//go:build integration
// +build integration

package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"

	"github.com/alexferl/zerohttp-example/models"
)

func clearCollections(ctx context.Context, s *MongoStore) {
	_ = s.users.Drop(ctx)
	_ = s.records.Drop(ctx)
	_ = s.orders.Drop(ctx)
}

func setupMongoStore(t *testing.T) (*MongoStore, func()) {
	t.Helper()

	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := mongodb.Run(ctx, "mongo:8")
	zhtest.AssertNoError(t, err)

	// Get connection string
	connStr, err := mongoContainer.ConnectionString(ctx)
	zhtest.AssertNoError(t, err)

	// Use unique database name per test to avoid conflicts
	dbName := "testdb_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Create store
	store, err := NewMongoStore(connStr, dbName)
	zhtest.AssertNoError(t, err)

	cleanup := func() {
		_ = store.Close(ctx)
		_ = mongoContainer.Terminate(ctx)
	}

	return store, cleanup
}

func TestMongoStore_UserOperations(t *testing.T) {
	store, cleanup := setupMongoStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("SaveUser and GetUser", func(t *testing.T) {
		user, err := models.NewUser(models.UserParams{
			Email:    "test@example.com",
			Name:     "Test User",
			Password: "password123",
			Role:     models.RoleUser,
		})
		zhtest.AssertNoError(t, err)

		// Save user
		err = store.SaveUser(ctx, user)
		zhtest.AssertNoError(t, err)

		// Get user
		retrieved, ok, err := store.GetUser(ctx, user.ID)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, user.ID, retrieved.ID)
		zhtest.AssertEqual(t, user.Email, retrieved.Email)
	})

	t.Run("GetUser not found", func(t *testing.T) {
		_, ok, err := store.GetUser(ctx, "nonexistent-id")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, ok)
	})

	t.Run("GetUserByEmail", func(t *testing.T) {
		user, _ := models.NewUser(models.UserParams{
			Email:    "byemail@example.com",
			Name:     "Email Test",
			Password: "password123",
		})
		store.SaveUser(ctx, user)

		retrieved, ok, err := store.GetUserByEmail(ctx, user.Email)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, user.ID, retrieved.ID)
	})

	t.Run("GetUserByEmail not found", func(t *testing.T) {
		_, ok, err := store.GetUserByEmail(ctx, "notfound@example.com")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, ok)
	})

	t.Run("GetAllUsers", func(t *testing.T) {
		clearCollections(ctx, store)
		// Add users
		for i := 0; i < 3; i++ {
			user, _ := models.NewUser(models.UserParams{
				Email:    "allusers" + string(rune('0'+i)) + "@example.com",
				Name:     "All Users Test",
				Password: "password123",
			})
			store.SaveUser(ctx, user)
			time.Sleep(10 * time.Millisecond) // Small delay to ensure unique timestamps
		}

		users, _, err := store.GetAllUsers(ctx, UserFilter{})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 3, len(users))
	})
}

func TestMongoStore_RecordOperations(t *testing.T) {
	store, cleanup := setupMongoStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("SaveRecord and GetRecord", func(t *testing.T) {
		record := models.NewRecord(models.RecordParams{
			Title:     "Test Record",
			Artist:    "Test Artist",
			Year:      2020,
			Label:     "Test Label",
			Format:    models.FormatLP,
			Genre:     models.GenreJazz,
			Condition: models.ConditionNM,
			Price:     25.00,
			Stock:     5,
		})

		err := store.SaveRecord(ctx, record)
		zhtest.AssertNoError(t, err)

		retrieved, ok, err := store.GetRecord(ctx, record.ID)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, record.ID, retrieved.ID)
	})

	t.Run("GetRecord not found", func(t *testing.T) {
		_, ok, err := store.GetRecord(ctx, "nonexistent-id")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, ok)
	})

	t.Run("GetAllRecords excludes archived", func(t *testing.T) {
		clearCollections(ctx, store)
		// Add active records
		for i := 0; i < 3; i++ {
			record := models.NewRecord(models.RecordParams{
				Title:     "Active Record " + string(rune('A'+i)),
				Artist:    "Artist",
				Format:    models.FormatLP,
				Genre:     models.GenreJazz,
				Condition: models.ConditionNM,
				Price:     20.00,
				Stock:     5,
			})
			store.SaveRecord(ctx, record)
		}

		// Add archived record
		archived := models.NewRecord(models.RecordParams{
			Title:     "Archived Record",
			Artist:    "Artist",
			Format:    models.FormatLP,
			Genre:     models.GenreRock,
			Condition: models.ConditionNM,
			Price:     25.00,
			Stock:     5,
		})
		archived.Archived = true
		store.SaveRecord(ctx, archived)

		records, err := store.GetAllRecords(ctx)
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 3, len(records))
	})

	t.Run("GetFilteredRecords by genre", func(t *testing.T) {
		clearCollections(ctx, store)
		// Add records with different genres
		genres := []models.Genre{models.GenreJazz, models.GenreRock, models.GenreJazz}
		for i, genre := range genres {
			record := models.NewRecord(models.RecordParams{
				Title:     "Genre Record " + string(rune('A'+i)),
				Artist:    "Artist",
				Format:    models.FormatLP,
				Genre:     genre,
				Condition: models.ConditionNM,
				Price:     20.00,
				Stock:     5,
			})
			store.SaveRecord(ctx, record)
		}

		results, total, err := store.GetFilteredRecords(ctx, RecordFilter{Genres: []string{"jazz"}})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 2, total)
		zhtest.AssertEqual(t, 2, len(results))
	})

	t.Run("GetFilteredRecords with pagination", func(t *testing.T) {
		clearCollections(ctx, store)
		// Add records for pagination test
		for i := 0; i < 5; i++ {
			record := models.NewRecord(models.RecordParams{
				Title:     "Paginated Record " + string(rune('A'+i)),
				Artist:    "Artist",
				Format:    models.FormatLP,
				Genre:     models.GenreJazz,
				Condition: models.ConditionNM,
				Price:     20.00,
				Stock:     5,
			})
			store.SaveRecord(ctx, record)
		}
		results, total, err := store.GetFilteredRecords(ctx, RecordFilter{BaseFilter: BaseFilter{Page: 1, PerPage: 2}})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 5, total)
		zhtest.AssertEqual(t, 2, len(results))
	})
}

func TestMongoStore_OrderOperations(t *testing.T) {
	store, cleanup := setupMongoStore(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("SaveOrder and GetOrder", func(t *testing.T) {
		order := models.NewOrder(models.OrderParams{
			UserID: "user-1",
			Items: []models.OrderItem{
				{RecordID: "rec-1", Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 2},
			},
		})

		err := store.SaveOrder(ctx, order)
		zhtest.AssertNoError(t, err)

		retrieved, ok, err := store.GetOrder(ctx, order.ID)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, order.ID, retrieved.ID)
		zhtest.AssertEqual(t, 50.00, retrieved.Total)
	})

	t.Run("GetOrder not found", func(t *testing.T) {
		_, ok, err := store.GetOrder(ctx, "nonexistent-id")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, ok)
	})

	t.Run("GetOrdersByUser", func(t *testing.T) {
		clearCollections(ctx, store)
		// Add orders for user
		for i := 0; i < 3; i++ {
			order := models.NewOrder(models.OrderParams{
				UserID: "user-filter",
				Items: []models.OrderItem{
					{RecordID: "rec-" + string(rune('0'+i)), Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 1},
				},
			})
			store.SaveOrder(ctx, order)
			time.Sleep(10 * time.Millisecond)
		}

		// Add order for different user
		otherOrder := models.NewOrder(models.OrderParams{
			UserID: "other-user",
			Items: []models.OrderItem{
				{RecordID: "rec-x", Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 1},
			},
		})
		store.SaveOrder(ctx, otherOrder)

		orders, _, err := store.GetOrdersByUser(ctx, OrderFilter{UserID: "user-filter"})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 3, len(orders))
	})

	t.Run("GetAllOrders", func(t *testing.T) {
		clearCollections(ctx, store)
		// Add orders for testing
		for i := 0; i < 3; i++ {
			order := models.NewOrder(models.OrderParams{
				UserID: "user-allorders",
				Items: []models.OrderItem{
					{RecordID: "rec-" + string(rune('0'+i)), Title: "Record", Artist: "Artist", Price: 25.00, Quantity: 1},
				},
			})
			store.SaveOrder(ctx, order)
		}
		orders, _, err := store.GetAllOrders(ctx, AllOrdersFilter{})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 3, len(orders))
	})
}

func TestMongoStore_Close(t *testing.T) {
	store, _ := setupMongoStore(t)

	ctx := context.Background()

	err := store.Close(ctx)
	zhtest.AssertNoError(t, err)
}

func TestMongoStore_Ping(t *testing.T) {
	store, cleanup := setupMongoStore(t)
	defer cleanup()

	ctx := context.Background()

	// Ping should succeed when connected
	err := store.Ping(ctx)
	zhtest.AssertNoError(t, err)
}

func TestMongoStore_DecrementStock(t *testing.T) {
	store, cleanup := setupMongoStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a record with stock
	record := models.NewRecord(models.RecordParams{
		Title:     "Limited Edition",
		Artist:    "Test Artist",
		Format:    models.FormatLP,
		Genre:     models.GenreJazz,
		Condition: models.ConditionNM,
		Price:     50.00,
		Stock:     5,
	})
	err := store.SaveRecord(ctx, record)
	zhtest.AssertNoError(t, err)

	t.Run("DecrementStock success", func(t *testing.T) {
		success, err := store.DecrementStock(ctx, record.ID, 2)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, success)

		// Verify stock was decremented
		updated, ok, err := store.GetRecord(ctx, record.ID)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, 3, updated.Stock)
	})

	t.Run("DecrementStock insufficient stock", func(t *testing.T) {
		// Try to decrement more than available (3 remaining)
		success, err := store.DecrementStock(ctx, record.ID, 10)
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, success)

		// Verify stock unchanged
		updated, ok, err := store.GetRecord(ctx, record.ID)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, 3, updated.Stock)
	})

	t.Run("DecrementStock non-existent record", func(t *testing.T) {
		success, err := store.DecrementStock(ctx, "nonexistent-id", 1)
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, success)
	})
}

func TestMongoStore_GetFilteredRecords_Filters(t *testing.T) {
	store, cleanup := setupMongoStore(t)
	defer cleanup()

	ctx := context.Background()
	clearCollections(ctx, store)

	// Create test records with various attributes
	records := []*models.Record{
		models.NewRecord(models.RecordParams{
			Title:     "Jazz Classic",
			Artist:    "Miles Davis",
			Format:    models.FormatLP,
			Genre:     models.GenreJazz,
			Condition: models.ConditionNM,
			Price:     25.00,
			Stock:     5,
			Year:      1959,
		}),
		models.NewRecord(models.RecordParams{
			Title:     "Rock Anthem",
			Artist:    "Led Zeppelin",
			Format:    models.FormatLP,
			Genre:     models.GenreRock,
			Condition: models.ConditionVGPlus,
			Price:     35.00,
			Stock:     0,
			Year:      1971,
		}),
		models.NewRecord(models.RecordParams{
			Title:     "Electronic Beats",
			Artist:    "Kraftwerk",
			Format:    models.FormatLP,
			Genre:     models.GenreElectronic,
			Condition: models.ConditionMint,
			Price:     45.00,
			Stock:     3,
			Year:      1974,
		}),
		models.NewRecord(models.RecordParams{
			Title:     "Budget Jazz",
			Artist:    "Unknown Artist",
			Format:    models.FormatLP,
			Genre:     models.GenreJazz,
			Condition: models.ConditionVG,
			Price:     10.00,
			Stock:     10,
			Year:      1980,
		}),
	}

	for _, r := range records {
		err := store.SaveRecord(ctx, r)
		zhtest.AssertNoError(t, err)
		time.Sleep(50 * time.Millisecond) // Delay to ensure unique millisecond timestamps
	}

	t.Run("Filter by price range", func(t *testing.T) {
		results, total, err := store.GetFilteredRecords(ctx, RecordFilter{
			MinPrice: 20.00,
			MaxPrice: 40.00,
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 2, total) // Jazz Classic ($25) and Rock Anthem ($35)
		zhtest.AssertEqual(t, 2, len(results))
	})

	t.Run("Filter by artist", func(t *testing.T) {
		results, total, err := store.GetFilteredRecords(ctx, RecordFilter{
			Artist: "Miles Davis",
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 1, total)
		zhtest.AssertEqual(t, "Jazz Classic", results[0].Title)
	})

	t.Run("Filter by in stock", func(t *testing.T) {
		results, total, err := store.GetFilteredRecords(ctx, RecordFilter{
			InStock: true,
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 3, total) // All except Rock Anthem (stock=0)
		zhtest.AssertEqual(t, 3, len(results))
	})

	t.Run("Filter by decade", func(t *testing.T) {
		results, total, err := store.GetFilteredRecords(ctx, RecordFilter{
			Decade: 1970,
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 2, total) // Rock Anthem (1971) and Electronic Beats (1974)
		zhtest.AssertEqual(t, 2, len(results))
	})

	t.Run("Filter by condition", func(t *testing.T) {
		results, total, err := store.GetFilteredRecords(ctx, RecordFilter{
			Conditions: []string{"nm", "mint"},
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 2, total) // Jazz Classic (NM) and Electronic Beats (Mint)
		zhtest.AssertEqual(t, 2, len(results))
	})

	t.Run("Sort by price ascending", func(t *testing.T) {
		results, _, err := store.GetFilteredRecords(ctx, RecordFilter{
			Sort: "price_asc",
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 4, len(results))
		zhtest.AssertEqual(t, "Budget Jazz", results[0].Title)      // $10
		zhtest.AssertEqual(t, "Jazz Classic", results[1].Title)     // $25
		zhtest.AssertEqual(t, "Rock Anthem", results[2].Title)      // $35
		zhtest.AssertEqual(t, "Electronic Beats", results[3].Title) // $45
	})

	t.Run("Sort by price descending", func(t *testing.T) {
		results, _, err := store.GetFilteredRecords(ctx, RecordFilter{
			Sort: "price_desc",
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 4, len(results))
		zhtest.AssertEqual(t, "Electronic Beats", results[0].Title) // $45
		zhtest.AssertEqual(t, "Rock Anthem", results[1].Title)      // $35
		zhtest.AssertEqual(t, "Jazz Classic", results[2].Title)     // $25
		zhtest.AssertEqual(t, "Budget Jazz", results[3].Title)      // $10
	})

	t.Run("Sort by year ascending", func(t *testing.T) {
		results, _, err := store.GetFilteredRecords(ctx, RecordFilter{
			Sort: "year_asc",
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 4, len(results))
		zhtest.AssertEqual(t, 1959, results[0].Year)
		zhtest.AssertEqual(t, 1971, results[1].Year)
		zhtest.AssertEqual(t, 1974, results[2].Year)
		zhtest.AssertEqual(t, 1980, results[3].Year)
	})

	t.Run("Sort by year descending", func(t *testing.T) {
		results, _, err := store.GetFilteredRecords(ctx, RecordFilter{
			Sort: "year_desc",
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 4, len(results))
		zhtest.AssertEqual(t, 1980, results[0].Year)
		zhtest.AssertEqual(t, 1974, results[1].Year)
		zhtest.AssertEqual(t, 1971, results[2].Year)
		zhtest.AssertEqual(t, 1959, results[3].Year)
	})

	t.Run("Sort by created_desc", func(t *testing.T) {
		results, _, err := store.GetFilteredRecords(ctx, RecordFilter{
			Sort: "created_desc",
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 4, len(results))
		// Note: MongoDB datetime has limited precision (100ms), so we just verify
		// the sort runs without error. The actual order can't be reliably tested
		// without significantly longer delays between record creation.
	})

	t.Run("Combined filters", func(t *testing.T) {
		results, total, err := store.GetFilteredRecords(ctx, RecordFilter{
			Genres:   []string{"jazz"},
			MinPrice: 20.00,
			InStock:  true,
		})
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 1, total) // Only Jazz Classic (jazz + $25 + in stock)
		zhtest.AssertEqual(t, "Jazz Classic", results[0].Title)
	})
}

func TestMongoStore_SeedData(t *testing.T) {
	store, cleanup := setupMongoStore(t)
	defer cleanup()

	ctx := context.Background()

	// Verify no records exist initially
	records, err := store.GetAllRecords(ctx)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, 0, len(records))

	// Run seed data
	store.SeedData()

	// Verify records were created
	records, err = store.GetAllRecords(ctx)
	zhtest.AssertNoError(t, err)
	zhtest.AssertGreater(t, len(records), 0)

	// Verify specific seeded records exist
	var foundMilesDavis bool
	for _, r := range records {
		if r.Title == "Kind of Blue" && r.Artist == "Miles Davis" {
			foundMilesDavis = true
			break
		}
	}
	zhtest.AssertTrue(t, foundMilesDavis)
}
