package product

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Handler struct {
	service *Service
	logger  zerolog.Logger
}

func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	moduleLogger := logger.With().Str("module", "product").Logger()
	return &Handler{
		service: service,
		logger:  moduleLogger,
	}
}

func (h *Handler) Create(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// parsear y validar el body JSON
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid data",
			"details": err.Error(),
		})
		return
	}

	// llamar al servicio
	product, err := h.service.Create(ctx, req)
	if err != nil {
		// manejar errores de negocio
		if strings.Contains(err.Error(), "already exists") {
			h.logger.Warn().Err(err).Str("sku", req.SKU).Msg("Attempt to create duplicate product")
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}

		h.logger.Error().Err(err).Str("sku", req.SKU).Msg("Error creating product")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error creating product",
		})
		return
	}
	h.logger.Info().Int("product_id", product.ID).Str("sku", product.SKU).Msg("Product created successfully")
	c.JSON(http.StatusCreated, product)
}

// GetByID maneja GET /products/:id
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Warn().
			Str("id_param", c.Param("id")).
			Msg("Invalid product ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		// Distinguir entre "no encontrado" y "error del servidor"
		if err.Error() == "product not found" {
			// No loggear 404 (es normal, ya lo loggea el middleware)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		h.logger.Error().
			Err(err).
			Int("product_id", id).
			Msg("Failed to fetch product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *Handler) GetBySKU(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	sku := c.Param("sku")
	if sku == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "SKU is required",
		})
		return
	}

	product, err := h.service.GetBySKU(ctx, sku)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		h.logger.Error().Err(err).Str("sku", sku).Msg("Error getting product by SKU")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting product by SKU",
		})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *Handler) GetAll(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	response, err := h.service.GetAll(ctx, page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).Int("page", page).Int("page_size", pageSize).Msg("Error getting all products")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting all products",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response.Products,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       response.Total,
			"total_pages": (response.Total + pageSize - 1) / pageSize,
		},
	})
}

func (h *Handler) Update(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid ID, must be a number",
		})
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Int("product_id", id).Msg("Invalid request body for update")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid data",
			"details": err.Error(),
		})
		return
	}

	product, err := h.service.Update(ctx, id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}

		h.logger.Error().Err(err).Int("product_id", id).Msg("Error updating product")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error updating product",
		})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *Handler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		h.logger.Warn().Err(err).Str("id_param", idParam).Msg("Invalid product ID format for delete")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid ID, must be a number",
		})
		return
	}

	err = h.service.Delete(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}

		h.logger.Error().Err(err).Int("product_id", id).Msg("Error deleting product")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error deleting product",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) SoftDelete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Warn().Err(err).Str("id_param", idStr).Msg("Invalid product ID format for get deleted")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid ID, must be a number",
		})
		return
	}

	err = h.service.SoftDelete(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		h.logger.Error().Err(err).Int("product_id", id).Msg("Failed to soft delete product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to soft delete product"})
		return
	}

	h.logger.Info().Int("product_id", id).Msg("Product soft deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Product soft deleted successfully"})
}

func (h *Handler) Restore(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Warn().Err(err).Str("id_param", idStr).Msg("Invalid product ID format for restore")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid ID, must be a number",
		})
		return
	}

	err = h.service.Restore(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		h.logger.Error().Err(err).Int("product_id", id).Msg("Failed to restore product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore product"})
		return
	}

	h.logger.Info().Int("product_id", id).Msg("Product restored successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Product restored successfully"})
}

func (h *Handler) GetAllDeleted(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	response, err := h.service.GetDeleted(ctx, page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get deleted products")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve deleted products"})
	}

	h.logger.Info().Int("page", page).Int("page_size", pageSize).Msg("Deleted products retrieved successfully")
	c.JSON(http.StatusOK, response)
}

func (h *Handler) Search(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var filters SearchFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid query parameters for search")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := h.service.Search(ctx, filters)
	if err != nil {
		if err.Error() == "invalid date format for from_date. expected YYYY-MM-DD" ||
			err.Error() == "invalid date format for to_date. expected YYYY-MM-DD" ||
			err.Error() == "from_date must be less than or equal to to_date" ||
			err.Error() == "min_stock cannot be negative" ||
			err.Error() == "max_stock cannot be negative" ||
			err.Error() == "min_stock cannot be greater than max_stock" ||
			err.Error() == "invalid price format for min_price. expected float" ||
			err.Error() == "invalid price format for max_price. expected float" ||
			err.Error() == "min_price must be less than or equal to max_price" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error().Err(err).Msg("Failed to search products")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search products"})
		return
	}

	h.logger.Info().
		Int("total", response.Pagination.Total).
		Int("results", len(response.Data.([]ProductResponse))).
		Msg("Products searched successfully")
	c.JSON(http.StatusOK, response)
}
