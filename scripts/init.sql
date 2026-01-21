-- ==============================================
-- WMS Lite - Inicialización de Base de Datos
-- ==============================================

-- 1. Crear ENUM para tipo de movimiento
CREATE TYPE movement_type AS ENUM ('IN', 'OUT');

-- 2. Tabla de Productos
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    sku VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índice para búsquedas por SKU
CREATE INDEX idx_products_sku ON products(sku);

-- 3. Tabla de Movimientos
CREATE TABLE IF NOT EXISTS movements (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    movement_type movement_type NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100)
);

-- Índices para consultas frecuentes
CREATE INDEX idx_movements_product_id ON movements(product_id);
CREATE INDEX idx_movements_created_at ON movements(created_at DESC);

-- 4. Trigger para actualizar updated_at automáticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Datos de prueba (opcional - comentar si no se necesita)
INSERT INTO products (sku, name, description, stock_quantity) VALUES
    ('LAPTOP-001', 'Laptop Dell XPS 15', 'Laptop profesional 15 pulgadas', 10),
    ('MOUSE-001', 'Mouse Logitech MX Master', 'Mouse ergonómico inalámbrico', 25),
    ('KEYBOARD-001', 'Teclado Mecánico Keychron K2', 'Teclado mecánico RGB', 0),
    ('MONITOR-001', 'Monitor Dell 27" 4K', 'Monitor UHD 27 pulgadas', 8)
ON CONFLICT (sku) DO NOTHING;

-- Mensaje de confirmación
DO $$
BEGIN
    RAISE NOTICE '✅ Base de datos inicializada correctamente';
END $$;