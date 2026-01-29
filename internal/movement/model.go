package movement

import "time"

// Representa el tipo de movimiento (IN OUT)
type MovementType string

const (
	MovementTypeIn  MovementType = "IN"
	MovementTypeOut MovementType = "OUT"
)

// isValid verifica si el movimiento es valido
func (mt MovementType) IsValid() bool {
	return mt == MovementTypeIn || mt == MovementTypeOut
}

type Movement struct {
	ID           int          `db:"id" json:"id"`
	ProductID    int          `db:"product_id" json:"product_id"`
	MovementType MovementType `db:"movement_type" json:"movement_type"`
	Quantity     int          `db:"quantity" json:"quantity"`
	Reason       string       `db:"reason" json:"reason"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
	CreatedBy    string       `db:"created_by" json:"created_by"`
}

type CreateMovementRequest struct {
	ProductID    int          `json:"product_id" binding:"required,min=1"`
	MovementType MovementType `json:"movement_type" binding:"required"`
	Quantity     int          `json:"quantity" binding:"required,min=1"`
	Reason       string       `json:"reason" binding:"max=255"`
	CreatedBy    string       `json:"created_by" binding:"max=100"`
}

type MovementResponse struct {
	ID           int          `json:"id"`
	ProductID    int          `json:"product_id"`
	MovementType MovementType `json:"movement_type"`
	Quantity     int          `json:"quantity"`
	Reason       string       `json:"reason,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	CreatedBy    string       `json:"created_by,omitempty"`
}

type ListMovementResponse struct {
	Data       []MovementResponse `json:"data"`
	Pagination PaginationMeta     `json:"pagination"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// convierte un movimiento en un movimiento response
func (m *Movement) ToResponse() MovementResponse {
	return MovementResponse{
		ID:           m.ID,
		ProductID:    m.ProductID,
		MovementType: m.MovementType,
		Quantity:     m.Quantity,
		Reason:       m.Reason,
		CreatedAt:    m.CreatedAt,
		CreatedBy:    m.CreatedBy,
	}
}
