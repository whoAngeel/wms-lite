package product

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

type Repository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

func NewRepository(db *sqlx.DB, logger zerolog.Logger) *Repository {
	moduleLogger := logger.With().Str("module", "product").Logger()
	return &Repository{db: db, logger: moduleLogger}
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
		WHERE id = $1 AND deleted_at IS NULL
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
		WHERE sku = $1 AND deleted_at IS NULL
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
		WHERE deleted_at IS NULL
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
	query := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`

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

func (r *Repository) SoftDelete(ctx context.Context, id int) error {
	query := `
		UPDATE products 
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Int("product_id", id).Msg("Failed to soft delete product")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		r.logger.Warn().Int("product_id", id).Msg("Product not found or already deleted")
		return fmt.Errorf("product not found or already deleted")
	}

	r.logger.Info().Int("product_id", id).Msg("Product soft deleted successfully")
	return nil

}

func (r *Repository) Restore(ctx context.Context, id int) error {
	query := `
		UPDATE products
		SET deleted_at = NULL
		WHERE id = $1 AND deleted_at IS NOT NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Int("product_id", id).Msg("Failed to restore product")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		r.logger.Warn().Int("product_id", id).Msg("Product not found or already active")
		return fmt.Errorf("product not found or already active")
	}

	r.logger.Info().Int("product_id", id).Msg("Product restored successfully")
	return nil
}

func (r *Repository) GetDeleted(ctx context.Context, page, pageSize int) ([]Product, int, error) {
	offset := (page - 1) * pageSize

	query := `
		SELECT id, sku, name, description, stock_quantity, created_at, updated_at, deleted_at
		FROM products
		WHERE deleted_at IS NOT NULL
		ORDER BY deleted_at DESC
		LIMIT $1 OFFSET $2
	`

	var products []Product
	err := r.db.SelectContext(ctx, &products, query, pageSize, offset)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get deleted products")
		return nil, 0, err
	}

	var total int
	countQuery := `
		SELECT COUNT(*) FROM products WHERE deleted_at IS NOT NULL`

	err = r.db.GetContext(ctx, &total, countQuery)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to count deleted products")
		return nil, 0, err
	}

	r.logger.Info().Int("page", page).Int("pageSize", pageSize).Int("total", total).Msg("Deleted products retrieved successfully")

	return products, total, nil

}

func (r *Repository) Search(ctx context.Context, filters SearchFilters) ([]Product, int, error) {
	// query base
	baseQuery := `
		SELECT id, sku, name, description, stock_quantity, created_at, updated_at, deleted_at
		FROM products
		WHERE deleted_at IS NULL
	`
	var conditions []string
	var args []interface{}
	argPosition := 1

	if filters.Name != "" {
		conditions = append(conditions, fmt.Sprintf(" name ILIKE '%%' || $%d || '%%'", argPosition))
		args = append(args, filters.Name)
		argPosition++
	}

	if filters.SKU != "" {
		conditions = append(conditions, fmt.Sprintf(" sku ILIKE '%%' || $%d || '%%'", argPosition))
		args = append(args, filters.SKU)
		argPosition++
	}

	if filters.MinStock != nil {
		conditions = append(conditions, fmt.Sprintf(" stock_quantity >= $%d", argPosition))
		args = append(args, *filters.MinStock)
		argPosition++
	}

	if filters.MaxStock != nil {
		conditions = append(conditions, fmt.Sprintf(" stock_quantity <= $%d", argPosition))
		args = append(args, *filters.MaxStock)
		argPosition++
	}

	if filters.FromDate != "" {
		conditions = append(conditions, fmt.Sprintf(" created_at >= $%d", argPosition))
		args = append(args, filters.FromDate)
		argPosition++
	}

	if filters.ToDate != "" {
		conditions = append(conditions, fmt.Sprintf(
			" created_at <= $%d::date + INTERVAL '23 hours 59 minutes 59 seconds'",
			argPosition,
		))
		args = append(args, filters.ToDate)
		argPosition++
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY created_at DESC"

	// contar total antes de aplicar limit y offser
	countQuery := "SELECT COUNT(*) FROM products WHERE deleted_at IS NULL "
	if len(conditions) > 0 {
		countQuery += " AND " + strings.Join(conditions, " AND ")
	}

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to count search results")
		return nil, 0, err
	}

	// agregar pagination
	offset := (filters.Page - 1) * filters.PageSize
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPosition, argPosition+1)
	args = append(args, filters.PageSize, offset)

	var products []Product
	err = r.db.SelectContext(ctx, &products, baseQuery, args...)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to search products")
		return nil, 0, err
	}

	r.logger.Info().
		Int("page", filters.Page).
		Int("pageSize", filters.PageSize).
		Int("total", total).
		Msg("Products searched successfully")

	return products, total, nil
}
