package models

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestNewUser_Success(t *testing.T) {
	user, err := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     RoleUser,
	})

	zhtest.AssertNoError(t, err)
	zhtest.AssertNotNil(t, user)
	zhtest.AssertEqual(t, "test@example.com", user.Email)
	zhtest.AssertEqual(t, "Test User", user.Name)
	zhtest.AssertEqual(t, RoleUser, user.Role)
	zhtest.AssertTrue(t, user.Active)
	zhtest.AssertNotEmpty(t, user.ID)
	zhtest.AssertNotEmpty(t, user.PasswordHash)
}

func TestNewUser_DefaultRole(t *testing.T) {
	user, err := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
	})

	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, RoleUser, user.Role)
}

func TestNewUser_WithAddress(t *testing.T) {
	addr := Address{
		Street:  "123 Main St",
		City:    "San Francisco",
		State:   "CA",
		ZipCode: "94102",
		Country: "USA",
	}

	user, err := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Role:     RoleUser,
		Address:  addr,
	})

	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, addr, user.Address)
}

func TestNewUser_AdminRole(t *testing.T) {
	user, err := NewUser(UserParams{
		Email:    "admin@example.com",
		Name:     "Admin User",
		Password: "adminpass123",
		Role:     RoleAdmin,
	})

	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, RoleAdmin, user.Role)
}

func TestVerifyPassword_Correct(t *testing.T) {
	user, _ := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "mypassword",
		Role:     RoleUser,
	})

	zhtest.AssertTrue(t, user.VerifyPassword("mypassword"))
}

func TestVerifyPassword_Incorrect(t *testing.T) {
	user, _ := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "mypassword",
		Role:     RoleUser,
	})

	zhtest.AssertFalse(t, user.VerifyPassword("wrongpassword"))
}

func TestVerifyPassword_Empty(t *testing.T) {
	user, _ := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "mypassword",
		Role:     RoleUser,
	})

	zhtest.AssertFalse(t, user.VerifyPassword(""))
}

func TestNewUser_GeneratesUniqueIDs(t *testing.T) {
	user1, _ := NewUser(UserParams{
		Email:    "user1@example.com",
		Name:     "User One",
		Password: "password123",
	})

	user2, _ := NewUser(UserParams{
		Email:    "user2@example.com",
		Name:     "User Two",
		Password: "password123",
	})

	zhtest.AssertNotEqual(t, user1.ID, user2.ID)
}

func TestNewUser_SetsTimestamps(t *testing.T) {
	user, _ := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
	})

	zhtest.AssertFalse(t, user.CreatedAt.IsZero())
	zhtest.AssertFalse(t, user.UpdatedAt.IsZero())
}

func TestUser_Update_NameOnly(t *testing.T) {
	user, _ := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Original Name",
		Password: "password123",
		Role:     RoleUser,
	})
	originalUpdatedAt := user.UpdatedAt

	user.Update("New Name", user.Address)

	zhtest.AssertEqual(t, "New Name", user.Name)
	zhtest.AssertTrue(t, user.UpdatedAt.After(originalUpdatedAt))
}

func TestUser_Update_AddressOnly(t *testing.T) {
	originalAddr := Address{Street: "123 Main St", City: "SF"}
	user, _ := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
		Address:  originalAddr,
	})
	originalName := user.Name
	originalUpdatedAt := user.UpdatedAt

	newAddr := Address{Street: "456 Oak St", City: "LA"}
	user.Update("", newAddr)

	zhtest.AssertEqual(t, originalName, user.Name)
	zhtest.AssertEqual(t, newAddr, user.Address)
	zhtest.AssertTrue(t, user.UpdatedAt.After(originalUpdatedAt))
}

func TestUser_Update_BothFields(t *testing.T) {
	user, _ := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Original Name",
		Password: "password123",
		Address:  Address{Street: "123 Main St"},
	})

	newAddr := Address{Street: "456 Oak St"}
	user.Update("New Name", newAddr)

	zhtest.AssertEqual(t, "New Name", user.Name)
	zhtest.AssertEqual(t, newAddr, user.Address)
}

func TestUser_Deactivate(t *testing.T) {
	user, _ := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
	})
	zhtest.AssertTrue(t, user.Active)
	originalUpdatedAt := user.UpdatedAt

	user.Deactivate()

	zhtest.AssertFalse(t, user.Active)
	zhtest.AssertTrue(t, user.UpdatedAt.After(originalUpdatedAt))
}

func TestUser_Deactivate_AlreadyInactive(t *testing.T) {
	user, _ := NewUser(UserParams{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
	})
	user.Active = false
	originalUpdatedAt := user.UpdatedAt

	user.Deactivate()

	zhtest.AssertFalse(t, user.Active)
	zhtest.AssertTrue(t, user.UpdatedAt.After(originalUpdatedAt))
}
