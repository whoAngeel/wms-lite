package movement

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Handler maneja las peticiones HTTP de movements
type Handler struct {
	service *Service
	logger  zerolog.Logger
}

// NewHandler crea una nueva instancia de Handler
func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	loggerModule := logger.With().Str("module", "movements").Logger()
	return &Handler{
		service: service,
		logger:  loggerModule,
	}
}

// Create maneja POST /movements
// @Summary Registrar un nuevo movimiento de inventario
// @Description Crea un movimiento de tipo IN (entrada) o OUT (salida) y actualiza el stock del producto
// @Tags movements
// @Accept json
// @Produce json
// @Param movement body CreateMovementRequest true "Datos del movimiento"
// @Success 201 {object} MovementResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /movements [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateMovementRequest

	// Parsear y validar JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalizar movement_type a mayúsculas
	req.MovementType = MovementType(strings.ToUpper(string(req.MovementType)))

	// Crear movimiento
	response, err := h.service.CreateMovement(c.Request.Context(), req)
	if err != nil {
		// Determinar el código de estado según el error
		statusCode := http.StatusInternalServerError
		errorMessage := err.Error()

		// Si es un error de validación o negocio, retornar 400
		if strings.Contains(errorMessage, "invalid") ||
			strings.Contains(errorMessage, "must be") ||
			strings.Contains(errorMessage, "insufficient stock") ||
			strings.Contains(errorMessage, "not found") {
			h.logger.Warn().Err(err).Int("product_id", req.ProductID).Str("movement_type", string(req.MovementType)).Msg("Business validation failed")
			statusCode = http.StatusBadRequest
		} else {
			h.logger.Error().Err(err).Int("product_id", req.ProductID).Str("movement_type", string(req.MovementType)).Msg("Error creating movement")
		}

		c.JSON(statusCode, gin.H{"error": errorMessage})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetByID maneja GET /movements/:id
// @Summary Obtener un movimiento por ID
// @Description Retorna los detalles de un movimiento específico
// @Tags movements
// @Produce json
// @Param id path int true "Movement ID"
// @Success 200 {object} MovementResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /movements/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	// Parsear ID del path parameter
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		h.logger.Warn().Str("id_param", idParam).Msg("Invalid movement ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movement ID"})
		return
	}

	// Obtener movimiento
	response, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == sql.ErrNoRows || strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else {
			h.logger.Error().Err(err).Int("movement_id", id).Msg("Error getting movement")
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListByProductID maneja GET /movements/product/:id
// @Summary Listar movimientos de un producto específico
// @Description Retorna todos los movimientos de un producto con paginación
// @Tags movements
// @Produce json
// @Param id path int true "Product ID"
// @Param page query int false "Número de página" default(1)
// @Param page_size query int false "Tamaño de página" default(10)
// @Success 200 {object} ListMovementsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /movements/product/{id} [get]
func (h *Handler) ListByProductID(c *gin.Context) {
	// Parsear product ID del path parameter
	idParam := c.Param("id")
	productID, err := strconv.Atoi(idParam)
	if err != nil {
		h.logger.Warn().Str("id_param", idParam).Msg("Invalid product ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}

	// Parsear parámetros de paginación
	page := h.parseIntQueryParam(c, "page", 1)
	pageSize := h.parseIntQueryParam(c, "page_size", 10)

	// Obtener movimientos
	response, err := h.service.ListByProductID(c.Request.Context(), productID, page, pageSize)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else {
			h.logger.Error().Err(err).Int("product_id", productID).Msg("Error listing movements by product")
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// List maneja GET /movements
// @Summary Listar movimientos con filtros opcionales
// @Description Retorna movimientos filtrados por product_id y/o movement_type con paginación
// @Tags movements
// @Produce json
// @Param page query int false "Número de página" default(1)
// @Param page_size query int false "Tamaño de página" default(10)
// @Param product_id query int false "Filtrar por ID de producto"
// @Param movement_type query string false "Filtrar por tipo (IN o OUT)"
// @Success 200 {object} ListMovementsResponse
// @Failure 400 {object} ErrorResponse
// @Router /movements [get]
func (h *Handler) List(c *gin.Context) {
	// Parsear parámetros de paginación
	page := h.parseIntQueryParam(c, "page", 1)
	pageSize := h.parseIntQueryParam(c, "page_size", 10)

	// CONCEPTO NUEVO: Parsear filtros opcionales
	// Si el query param no está presente, será nil
	var productID *int
	if productIDStr := c.Query("product_id"); productIDStr != "" {
		id, err := strconv.Atoi(productIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product_id parameter"})
			return
		}
		productID = &id // Asignar el puntero
	}

	var movementType *MovementType
	if movementTypeStr := c.Query("movement_type"); movementTypeStr != "" {
		// Normalizar a mayúsculas
		mt := MovementType(strings.ToUpper(movementTypeStr))

		// Validar que sea un valor válido
		if !mt.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movement_type: must be IN or OUT"})
			return
		}

		movementType = &mt // Asignar el puntero
	}

	// Obtener movimientos con filtros opcionales
	response, err := h.service.List(c.Request.Context(), productID, movementType, page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).Msg("Error listing movements")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// parseIntQueryParam es un helper para parsear query params enteros con valor por defecto
func (h *Handler) parseIntQueryParam(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil || value < 1 {
		return defaultValue
	}

	return value
}

// ErrorResponse representa un error HTTP
type ErrorResponse struct {
	Error string `json:"error"`
}
