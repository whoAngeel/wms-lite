package product

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// inserta un nuevo producto
func (r *Repository) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
	query := `
		INSERT INTO products (sku, name, description, stock_quantity)
		VALUES ($1, $2, $3, $4)
		RETURNING id, sku, name, description, stock_quantity, created_at, updated_at
	`

	var product Product
	err := r.db.QueryRowContext(
		ctx, query, req.SKU, req.Name, req.Description, req.Stock,
	).Scan(&product.ID, &product.SKU, &product.Name, &product.Description, &product.Stock, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("error creating product: %w", err)

	}
	return &product, nil
}

func (r *Repository) GetByID(ctx context.Context, id int) (*Product, error) {
	query := `
		SELECT id, sku, name, description, stock_quantity, created_at, updated_at
		FROM products
		WHERE id = $1
	`

	var product Product
	err := r.db.GetContext(ctx, &product, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product with id [%d] not found", id)
		}
		return nil, fmt.Errorf("error getting product: %w", err)
	}
	return &product, nil
}

func (r *Repository) GetBySKU(ctx context.Context, sku string) (*Product, error) {
	query := `
		SELECT id, sku, name, description, stock_quantity, created_at, updated_at
		FROM products
		WHERE sku = $1
	`

	var product Product
	err := r.db.GetContext(ctx, &product, query, sku)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product with sku [%s] not found", sku)
		}
		return nil, fmt.Errorf("error getting product: %w", err)
	}
	return &product, nil
}

func (r *Repository) GetAll(ctx context.Context, limit, offset int) ([]Product, error) {
	query := `
		SELECT id, sku, name, description, stock_quantity, created_at, updated_at
		FROM products
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var products []Product
	err := r.db.SelectContext(ctx, &products, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error getting products: %w", err)
	}
	return products, nil
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM products`

	var count int
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("error counting products: %w", err)
	}
	return count, nil
}

func (r *Repository) Update(ctx context.Context, id int, req UpdateProductRequest) (*Product, error) {
	query := `
		UPDATE products
		SET name = $1, description = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id, sku, name, description, stock_quantity, created_at, updated_at	
	`

	var product Product
	err := r.db.QueryRowContext(ctx, query, req.Name, req.Description, id).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Description,
		&product.Stock,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product with id [%d] not found", id)
		}
		return nil, fmt.Errorf("error updating product: %w", err)
	}
	return &product, nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	query := `
		DELETE FROM products
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error deleting product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("product with id [%d] not found", id)
	}
	return nil
}
