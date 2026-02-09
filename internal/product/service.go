package product

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/whoAngeel/wms-lite/internal/platform"
)

type Service struct {
	repo   *Repository
	logger zerolog.Logger
	cache  *platform.Cache
}

func NewService(repo *Repository, logger zerolog.Logger, cache *platform.Cache) *Service {
	serviceLogger := logger.With().Str("service", "product").Logger()
	return &Service{repo: repo, logger: serviceLogger, cache: cache}
}

func (s *Service) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
	// 1 validaciones de negocio
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// 2 validacion de negocio sku unico
	existingProduct, err := s.repo.GetBySKU(ctx, req.SKU)
	if err == nil && existingProduct != nil {
		return nil, fmt.Errorf("sku [%s] already exists", req.SKU)
	}

	// si el error es no encontrado, continuamos

	// 3 crear el producto
	product, err := s.repo.Create(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("sku [%s] already exists", req.SKU)
		}
		return nil, fmt.Errorf("error creating product: %w", err)
	}

	return product, nil
}

func (s *Service) GetByID(ctx context.Context, id int) (*Product, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid ID: must be greater than 0")
	}

	cacheKey := fmt.Sprintf("product:%d", id)
	// buscar en cache
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		// cache HIT - deserializar JSON
		var product Product
		jsonErr := json.Unmarshal([]byte(cached), &product)
		if jsonErr == nil {
			s.logger.Debug().
				Int("product_id", id).
				Str("source", "cache").
				Msg("Product retrieved from cache")
			return &product, nil
		}
		// Si falla unmarshal, continuar a DB
		s.logger.Warn().
			Int("product_id", id).
			Err(jsonErr).
			Msg("Failed to unmarshal cached product")
	} else if err != redis.Nil {
		// error real de redis
		s.logger.Warn().
			Int("product_id", id).
			Err(err).
			Msg("Redis error while getting product")
	}

	// cache MISS - leer de BD

	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// guardar en redis
	jsonData, _ := json.Marshal(product)
	if setErr := s.cache.Set(ctx, cacheKey, jsonData, 5*time.Minute); setErr != nil {
		s.logger.Warn().
			Int("product_id", id).
			Err(setErr).
			Msg("Failed to cache product")
	} else {
		s.logger.Debug().
			Int("product_id", id).
			Msg("Product cached successfully")
	}

	return product, nil
}

func (s *Service) GetBySKU(ctx context.Context, sku string) (*Product, error) {
	if sku == "" {
		return nil, fmt.Errorf("invalid SKU: must not be empty")
	}

	cacheKey := fmt.Sprintf("product:sku:%s", sku)

	// leer cachet attempt
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		var product Product
		jsonErr := json.Unmarshal([]byte(cached), &product)
		if jsonErr == nil {
			s.logger.Debug().Str("sku", sku).Str("source", "cache").Msg("Product retrieved from cached")
			return &product, nil
		}
		s.logger.Warn().Err(jsonErr).Msg("Failed to unmarshal cached product")

	} else if err != redis.Nil {
		s.logger.Warn().Str("sku", sku).Err(err).Msg("Redis error while getting product")
	}

	// cache miss
	s.logger.Debug().Str("sku", sku).Msg("Cache miss - reading from DB")
	product, err := s.repo.GetBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}

	// guardar en redis
	jsonData, _ := json.Marshal(product)
	if setErr := s.cache.Set(ctx, cacheKey, jsonData, 5*time.Minute); setErr != nil {
		s.logger.Warn().Str("sku", sku).Err(setErr).Msg("Failed to cache product")
	} else {
		s.logger.Debug().Str("sku", sku).Msg("Product cached successfully")
	}

	return product, nil
}

func (s *Service) GetAll(ctx context.Context, page, pageSize int) (*ProductListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// calcular offset
	offset := (page - 1) * pageSize

	products, err := s.repo.GetAll(ctx, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("error getting products: %w", err)
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("error counting products: %w", err)
	}
	responseList := make([]ProductResponse, len(products))
	for i, p := range products {
		responseList[i] = p.ToResponse()
	}

	return &ProductListResponse{
		Products: responseList,
		Total:    total,
	}, nil
}

func (s *Service) Update(ctx context.Context, id int, req UpdateProductRequest) (*Product, error) {
	// validar id
	if id <= 0 {
		return nil, fmt.Errorf("invalid ID: it must be greater than 0")
	}

	// validacion de negocio
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	existingProduct, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("product not found")
	}

	// actualizar en bd
	product, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("error updating product: %w", err)
	}

	// invalidar cache
	cacheKeys := []string{
		fmt.Sprintf("product:%d", id),
		fmt.Sprintf("product:sku:%s", existingProduct.SKU),
	}

	if delErr := s.cache.Del(ctx, cacheKeys...); delErr != nil {
		s.logger.Warn().Err(delErr).Msg("Failed to invalidate cache")
	}
	s.logger.Debug().
		Int("product_id", id).
		Strs("cache_keys", cacheKeys).
		Msg("Product updated successfully")

	return product, nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid ID: must be greater than 0")
	}
	// NOTE: validar si tiene movimientos asociados? > prevenir borrado
	// tiene stock > 0? -> prevenir borrado
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting product: %w", err)
	}

	return nil
}

func (s *Service) validateCreateRequest(req CreateProductRequest) error {
	// normalizar SKU (mayus, sin espacios)
	req.SKU = strings.TrimSpace(strings.ToUpper(req.SKU))

	// validar asociaciones de negocio
	if len(req.SKU) < 3 {
		return fmt.Errorf("the SKU must be at least 3 chars long")
	}

	if len(req.Name) < 3 {
		return fmt.Errorf("the name must be at least 3 chars long")
	}

	if req.Stock < 0 {
		return fmt.Errorf("the stock cannot be negative")
	}

	return nil
}

func (s *Service) validateUpdateRequest(req UpdateProductRequest) error {
	// si se proporciona un nombre, validar
	if req.Name != "" && len(req.Name) < 3 {
		return fmt.Errorf("the name must be at least 3 chars long")
	}

	// si se proporciona una descripcion, validar
	if req.Description != "" && len(req.Description) < 3 {
		return fmt.Errorf("the description must be at least 3 chars long")
	}

	return nil
}

func (s *Service) SoftDelete(ctx context.Context, id int) error {
	// validar id
	if id == 0 {
		return fmt.Errorf("invalid product id")
	}

	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// intentar hacer soft delete
	err = s.repo.SoftDelete(ctx, id)
	if err != nil {
		return err
	}

	// invalidar cache
	cacheKeys := []string{
		fmt.Sprintf("product:%d", id),
		fmt.Sprintf("product:sku:%s", product.SKU),
	}

	if delErr := s.cache.Del(ctx, cacheKeys...); delErr != nil {
		s.logger.Warn().Err(delErr).Msg("Failed to invalidate cache")
	}
	s.logger.Debug().
		Int("product_id", id).
		Strs("cache_keys", cacheKeys).
		Msg("Product soft deleted successfully")

	return nil
}

func (s *Service) Restore(ctx context.Context, id int) error {
	if id == 0 {
		return fmt.Errorf("invalid product id")
	}

	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.repo.Restore(ctx, id)
	if err != nil {
		return err
	}

	// invalidar cache
	cacheKeys := []string{
		fmt.Sprintf("product:%d", id),
		fmt.Sprintf("product:sku:%s", product.SKU),
	}

	if delErr := s.cache.Del(ctx, cacheKeys...); delErr != nil {
		s.logger.Warn().Err(delErr).Msg("Failed to invalidate cache")
	}
	s.logger.Debug().
		Int("product_id", id).
		Strs("cache_keys", cacheKeys).
		Msg("Product restored successfully")

	return nil
}

func (s *Service) GetDeleted(ctx context.Context, page, pageSize int) (*PaginatedResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	products, total, err := s.repo.GetDeleted(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	var responses []DeletedProductResponse
	for _, product := range products {
		responses = append(responses, product.ToDeletedResponse())
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := &PaginatedResponse{
		Data: responses,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
			Total:      total,
		},
	}

	return response, nil

}

func (s *Service) Search(
	ctx context.Context, filters SearchFilters,
) (*PaginatedResponse, error) {
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.PageSize <= 0 {
		filters.PageSize = 10
	}

	filters.Name = strings.TrimSpace(filters.Name)
	filters.SKU = strings.TrimSpace(filters.SKU)
	filters.FromDate = strings.TrimSpace(filters.FromDate)
	filters.ToDate = strings.TrimSpace(filters.ToDate)

	// validar rango de stock (si ambos estan presentes)
	if filters.MinStock != nil && filters.MaxStock != nil {
		if *filters.MinStock > *filters.MaxStock {
			// TODO: logger invalid range min > max
			return nil, fmt.Errorf("min_stock cannot be greater than max_stock")
		}
	}

	// validar stock no sea negativo
	if filters.MinStock != nil && *filters.MinStock < 0 {
		return nil, fmt.Errorf("min_stock cannot be negative")
	}

	if filters.MaxStock != nil && *filters.MaxStock < 0 {
		return nil, fmt.Errorf("max_stock cannot be negative")
	}

	// validar rango de fechas (YYYY-MM-DD)
	if filters.FromDate != "" {
		if !isValidDateFormat(filters.FromDate) {
			return nil, fmt.Errorf("invalid date format for from_date. expected YYYY-MM-DD")
		}
	}

	if filters.ToDate != "" {
		if !isValidDateFormat(filters.ToDate) {
			return nil, fmt.Errorf("invalid date format for to_date. expected YYYY-MM-DD")
		}
	}

	// llamar al repository
	products, total, err := s.repo.Search(ctx, filters)
	if err != nil {
		return nil, err
	}

	var responses []ProductResponse
	for _, p := range products {
		responses = append(responses, p.ToResponse())
	}

	totalPages := (total + filters.PageSize - 1) / filters.PageSize

	response := &PaginatedResponse{
		Data: responses,
		Pagination: Pagination{
			Page:       filters.Page,
			PageSize:   filters.PageSize,
			TotalPages: totalPages,
			Total:      total,
		},
	}

	return response, nil

}
func isValidDateFormat(date string) bool {
	// Regex simple para validar formato YYYY-MM-DD
	if len(date) != 10 {
		return false
	}

	// Intentar parsear con time.Parse
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}
