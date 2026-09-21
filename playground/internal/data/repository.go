package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type ProductRepository struct {
	DB *sql.DB
}

// FindByID streams a single record out of MySQL using bounded contexts
func (repo *ProductRepository) FindByID(ctx context.Context, id int) (*Product, error) {
	query := `SELECT id, name, price, created_at FROM products WHERE id = ? LIMIT 1`

	var p Product
	// The database query honors the context timeout automatically
	err := repo.DB.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.Name, &p.Price, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("product record not found")
		}
		return nil, err
	}
	return &p, nil
}

// CreateWithAudit executes an ACID transaction to write a product and log an entry simultaneously
func (repo *ProductRepository) CreateWithAudit(ctx context.Context, name string, price float64) (*Product, error) {
	tx, err := repo.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // Safely triggers a rollback if any operational errors occur below

	// 1. Insert the new product record
	insertQuery := `INSERT INTO products (name, price, created_at) VALUES (?, ?, ?)`
	createdAt := time.Now()
	res, err := tx.ExecContext(ctx, insertQuery, name, price, createdAt)
	if err != nil {
		return nil, err
	}

	productID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	// 2. Write a simulated audit log history record inside the same transaction block
	auditQuery := `INSERT INTO audit_logs (action, target_id, timestamp) VALUES (?, ?, ?)`
	_, err = tx.ExecContext(ctx, auditQuery, "PRODUCT_CREATION", productID, createdAt)
	if err != nil {
		return nil, err // Both operations roll back cleanly
	}

	// 3. Commit the transaction to save changes permanently to the database
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Product{
		ID:        int(productID),
		Name:      name,
		Price:     price,
		CreatedAt: createdAt,
	}, nil
}
