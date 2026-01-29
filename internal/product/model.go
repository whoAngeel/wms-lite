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

type ProductListResponse struct {
	Products []Product `json:"products"`
	Total    int       `json:"total"`
}
