-- Migration: Add soft delete support to products
-- Date: 2026-02-05

ALTER TABLE products 
ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;

CREATE INDEX idx_products_deleted_at ON products(deleted_at) 
WHERE deleted_at IS NULL;

COMMENT ON COLUMN products.deleted_at IS 'Soft delete timestamp. NULL = active product';