package product

import (
	"context"
	"fmt"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
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

	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return product, nil
}

func (s *Service) GetBySKU(ctx context.Context, sku string) (*Product, error) {
	if sku == "" {
		return nil, fmt.Errorf("invalid SKU: must not be empty")
	}

	product, err := s.repo.GetBySKU(ctx, sku)
	if err != nil {
		return nil, err
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

	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("product not found")
	}

	// actualizar
	product, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("error updating product: %w", err)
	}

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
