package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User represents a row in shipman.users.
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserService exposes CRUD behaviour for users.
type UserService interface {
	Create(ctx context.Context, u *User) error
	Retrieve(ctx context.Context, id uuid.UUID) (User, error)
	RetrieveByEmail(ctx context.Context, email string) (User, error)
	List(ctx context.Context, limit, offset int) ([]User, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserRepository implements UserService using the package-level Pool.
type UserRepository struct{}

// NewUserRepository returns a repository.
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// Create inserts a new user and populates ID/CreatedAt/UpdatedAt on the struct.
func (repo *UserRepository) Create(ctx context.Context, u *User) error {
	const query = `
		INSERT INTO shipman.users (email, password_hash, full_name, role)
		VALUES ($1, $2, $3, COALESCE($4, 'user'))
		RETURNING id, created_at, updated_at
	`

	return Pool.QueryRowContext(ctx, query, u.Email, u.PasswordHash, u.FullName, u.Role).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

// Retrieve fetches a user by ID.
func (repo *UserRepository) Retrieve(ctx context.Context, id uuid.UUID) (User, error) {
	const query = `
		SELECT id, email, password_hash, full_name, role, created_at, updated_at
		FROM shipman.users
		WHERE id = $1
	`

	var u User
	err := Pool.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

// RetrieveByEmail fetches a user by email address.
func (repo *UserRepository) RetrieveByEmail(ctx context.Context, email string) (User, error) {
	const query = `
		SELECT id, email, password_hash, full_name, role, created_at, updated_at
		FROM shipman.users
		WHERE email = $1
	`

	var u User
	err := Pool.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

// List returns users ordered by newest first.
func (repo *UserRepository) List(ctx context.Context, limit, offset int) ([]User, error) {
	const query = `
		SELECT id, email, password_hash, full_name, role, created_at, updated_at
		FROM shipman.users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := Pool.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Update modifies the stored fields for a user.
func (repo *UserRepository) Update(ctx context.Context, u *User) error {
	const query = `
		UPDATE shipman.users
		SET email = $2,
			password_hash = $3,
			full_name = $4,
			role = $5,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	return Pool.QueryRowContext(ctx, query, u.ID, u.Email, u.PasswordHash, u.FullName, u.Role).
		Scan(&u.UpdatedAt)
}

// Delete removes a user by ID.
func (repo *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM shipman.users WHERE id = $1`
	_, err := Pool.ExecContext(ctx, query, id)
	return err
}
