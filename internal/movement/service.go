package movement

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/whoAngeel/wms-lite/internal/platform"
)

type Service struct {
	repo   *Repository
	db     *sqlx.DB
	cache  *platform.Cache
	logger zerolog.Logger
}

func NewService(repo *Repository, db *sqlx.DB, cache *platform.Cache, logger *zerolog.Logger) *Service {
	moduleLogger := logger.With().Str("module", "movement").Logger()
	return &Service{
		repo:   repo,
		cache:  cache,
		db:     db,
		logger: moduleLogger,
	}
}

// Crea un nuevo movimiento de inventario
// flujo
// 1. BEGIN transaction
// 2. SELECT stock for UPDATE (bloquea la fila)
// 3. Validar stock suficiente (si es OUT)
// 4. Calcular nuevo stock
// 5. UPDATE stock en products
// 6. INSERT movement
// 7. COMMIT (o ROLLBACK si hay error)
func (s *Service) CreateMovement(ctx context.Context, req CreateMovementRequest) (resp *MovementResponse, err error) {
	if err = s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	exists, err := s.repo.ProductExists(ctx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("error checking product existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("product with ID [%d] not found", req.ProductID)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}

	// patron critico: DEFER para rollback automatico
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// obtener stock actual con lock pesimista
	currentStock, sku, err := s.repo.GetProductStockForUpdate(ctx, tx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("error getting product stock: %w", err)
	}

	// calcular nuevo stock segun el tipo de movimiento
	var newStock int
	switch req.MovementType {
	case MovementTypeIn:
		newStock = currentStock + req.Quantity
	case MovementTypeOut:
		if currentStock < req.Quantity {
			return nil, fmt.Errorf("insufficient stock: available=%d, request=%d", currentStock, req.Quantity)
		}
		newStock = currentStock - req.Quantity
	}

	// actualizar stock del producto (dentro de la transaccion)
	err = s.repo.UpdateProductStock(ctx, tx, req.ProductID, newStock)
	if err != nil {
		return nil, fmt.Errorf("error updating product stock: %w", err)
	}

	movement := &Movement{
		ProductID:    req.ProductID,
		MovementType: req.MovementType,
		Quantity:     req.Quantity,
		Reason:       req.Reason,
		CreatedBy:    req.CreatedBy,
	}

	err = s.repo.Create(ctx, tx, movement)
	if err != nil {
		return nil, fmt.Errorf("error creating movement: %w", err)
	}

	// commit todo salio bien commit
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	// invalidar cache de producto afectado
	cacheKeys := []string{
		fmt.Sprintf("product:%d", req.ProductID),
		fmt.Sprintf("product:sku:%s", sku),
	}

	if err := s.cache.Del(ctx, cacheKeys...); err != nil {
		s.logger.Warn().Err(err).Int("product_id", req.ProductID).Msg("Failed to invalidate product cache")
	} else {
		s.logger.Debug().Int("product_id", req.ProductID).Msg("Product cache invalidated successfully")
	}

	// retornar el movimiento creado
	response := movement.ToResponse()
	return &response, nil
}

func (s *Service) GetByID(ctx context.Context, id int) (*MovementResponse, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid movement ID: %d", id)
	}

	movement, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("movement with ID %d not found", id)
		}
		return nil, fmt.Errorf("error getting movement: %w", err)

	}
	response := movement.ToResponse()
	return &response, nil
}

func (s *Service) ListByProductID(ctx context.Context, productID, page, pageSize int) (*ListMovementResponse, error) {
	if productID <= 0 {
		return nil, fmt.Errorf("invalid product ID: %d", productID)
	}
	page, pageSize = s.normalizePagination(page, pageSize)

	// virificar que el producto existe
	exists, err := s.repo.ProductExists(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("error checking product existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("product with ID [%d] not found", productID)
	}

	movements, total, err := s.repo.ListByProductID(ctx, productID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("error listing movements: %w", err)
	}

	responses := make([]MovementResponse, len(movements))
	for i, m := range movements {
		responses[i] = m.ToResponse()
	}

	// calcular total de paginas
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	return &ListMovementResponse{
		Data: responses,
		Pagination: PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *Service) List(ctx context.Context, productID *int, movementType *MovementType, page, pageSize int) (*ListMovementResponse, error) {
	// validaciones de filtros
	if productID != nil && *productID <= 0 {
		return nil, fmt.Errorf("invalid product ID: %d", *productID)
	}

	if movementType != nil && !movementType.IsValid() {
		return nil, fmt.Errorf("invalid movement type: %s", *movementType)
	}

	page, pageSize = s.normalizePagination(page, pageSize)

	// si se especifica productID, validar que existe
	if productID != nil {
		exists, err := s.repo.ProductExists(ctx, *productID)
		if err != nil {
			return nil, fmt.Errorf("error checking product existence: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("product with ID [%d] not found", *productID)
		}
	}

	movements, total, err := s.repo.List(ctx, productID, movementType, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("error listing movements: %w", err)
	}

	responses := make([]MovementResponse, len(movements))
	for i, m := range movements {
		responses[i] = m.ToResponse()
	}

	// calcular total de paginas
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	return &ListMovementResponse{
		Data: responses,
		Pagination: PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil

}

func (s *Service) validateCreateRequest(req CreateMovementRequest) error {
	if req.ProductID <= 0 {
		return fmt.Errorf("product_id must be greater than 0")
	}

	if !req.MovementType.IsValid() {
		return fmt.Errorf("invalid movement_type: must be 'IN' or 'OUT'")
	}

	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}

	if len(req.Reason) > 255 {
		return fmt.Errorf("reason must be at most 255 characters")
	}

	if len(req.CreatedBy) > 100 {
		return fmt.Errorf("created_by must be at most 100 characters")
	}
	return nil
}

func (s *Service) normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = 10
	}

	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
