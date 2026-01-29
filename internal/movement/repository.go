package movement

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, tx *sqlx.Tx, movement *Movement) error {
	query := `
		INSERT INTO movements (product_id, movement_type, quantity, reason, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := tx.QueryRowxContext(
		ctx, query, movement.ProductID, movement.MovementType, movement.Reason, movement.CreatedBy,
	).Scan(&movement.ID, &movement.CreatedAt)

	if err != nil {
		return fmt.Errorf("error creating movement: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int) (*Movement, error) {
	var movement Movement
	query := `
		SELECT id, product_id, movement_type, quantity, reason, created_at, created_by
		FROM movements
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &movement, query, id)
	if err != nil {
		return nil, fmt.Errorf("error getting movement by id: %w", err)
	}
	return &movement, nil
}

func (r *Repository) ListByProductID(ctx context.Context, productID, page, pageSize int) ([]Movement, int, error) {
	var movements []Movement
	offset := (page - 1) * pageSize
	query := `
		SELECT id, product, movement_type, quantity, reason, created_at, created_by
		FROM movements
		WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	err := r.db.SelectContext(ctx, &movements, query, productID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error listening movements by product: %w", err)
	}

	var total int
	countQuery := `
		SELECT COUNT(*) FROM movements WHERE product_id = $1
	`
	err = r.db.GetContext(ctx, &total, countQuery, productID)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting movements: %w", err)
	}

	return movements, total, nil
}

// obtiene los elementos con paginacion y filtros opcionales
func (r *Repository) List(ctx context.Context, productID *int, movementType *MovementType, page, pageSize int) ([]Movement, int, error) {
	var movements []Movement
	offset := (page - 1) * pageSize

	query := `
		SELECT id, product_id, movement_type, quantity, reason, created_at, created_by
		FROM movements
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM movements WHERE 1=1`

	// slice para los argumentos
	args := []interface{}{}
	argPosition := 1

	// agregar filtro de product_id si ya existe
	if productID != nil {
		query += fmt.Sprintf(" AND product_id = $%d", argPosition)
		countQuery += fmt.Sprintf(" AND product_id = $%d", argPosition)
		args = append(args, *productID)
		argPosition++
	}

	// agregar filtro de movement_type si existe
	if movementType != nil {
		query += fmt.Sprintf(" AND movement_type = $%d", argPosition)
		countQuery += fmt.Sprintf(" AND movement_type = $%d", argPosition)
		args = append(args, *movementType)
		argPosition++
	}

	// agegar order by limit y offset
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPosition, argPosition+1)
	args = append(args, pageSize, offset)

	// ejecutar query principal
	err := r.db.SelectContext(ctx, &movements, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error listing movements: %w", err)
	}

	// contar total
	var total int
	countArgs := args[:len(args)-2] // remover limit y offset del conteo
	err = r.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting movements: %w", err)
	}
	return movements, total, nil
}

// obtiene el stock actuan de un producto con LOCK PESIMISTA
// CONCEPTO CRITICO: SELECT ... FOR UPDATE
// ESTO bloquea la fila hasta que la transaccion termine (COMMIT o ROLLBACK)
// Previene race conditions cuando dos requests intentan modificar el mismo producto
func (r *Repository) GetProductStockForUpdate(ctx context.Context, tx *sqlx.Tx, productID int) (int, error) {
	var stock int
	query := `
		SELECT stock_quantity
		FROM products 
		WHERE id = $1
		FOR UPDATE
	`

	// DEBE ejecutarse dentro de una transaccion
	err := tx.QueryRowxContext(ctx, query, productID).Scan(&stock)
	if err != nil {
		return 0, fmt.Errorf("error getting product stock with lock: %w", err)
	}

	return stock, nil
}

// Actualiza el stock de un producto
// DEBE ejecutarse dentro de una transaccion junto con create()
func (r *Repository) UpdateProductStock(ctx context.Context, tx *sqlx.Tx, productID, newStock int) error {
	query := `
		UPDATE products 
		SET stock_quantity = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := tx.ExecContext(ctx, query, newStock, productID)
	if err != nil {
		return fmt.Errorf("error updating product stock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("product not found")
	}

	return nil
}

func (r *Repository) ProductExists(ctx context.Context, productID int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)`

	err := r.db.GetContext(ctx, &exists, query, productID)
	if err != nil {
		return false, fmt.Errorf("error checking product existence: %w", err)
	}

	return exists, nil
}
