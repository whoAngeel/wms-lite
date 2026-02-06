package product

import "time"

type Product struct {
	ID          int       `json:"id" db:"id"`
	SKU         string    `json:"sku" db:"sku"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Stock       int       `json:"stock_quantity" db:"stock_quantity"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// create productRequest es el payload para crear un producto
type CreateProductRequest struct {
	SKU         string `json:"sku" validate:"required,max=50"`
	Name        string `json:"name" validate:"required,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
	Stock       int    `json:"stock_quantity" binding:"min=0" validate:"required,min=0"`
}

// updateProductRequest
type UpdateProductRequest struct {
	Name        string `json:"name" validate:"omitempty,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
	// STOCK no se actualiza aca, solo desde movements
}

type ProductResponse struct {
	ID          int       `json:"id" db:"id"`
	SKU         string    `json:"sku" db:"sku"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Stock       int       `json:"stock_quantity" db:"stock_quantity"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// DeletedProductResponse DTO para productos eliminados (incluye deleted_at)
type DeletedProductResponse struct {
	ID          int       `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Stock       int       `json:"stock_quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at"`
}

type ProductListResponse struct {
	Products []ProductResponse `json:"products"`
	Total    int               `json:"total"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Converters
func (p *Product) ToResponse() ProductResponse {
	return ProductResponse{
		ID:          p.ID,
		SKU:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		Stock:       p.Stock,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// ToDeletedResponse convierte a respuesta para productos eliminados
func (p *Product) ToDeletedResponse() DeletedProductResponse {
	return DeletedProductResponse{
		ID:          p.ID,
		SKU:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		Stock:       p.Stock,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		DeletedAt:   p.DeletedAt,
	}
}
