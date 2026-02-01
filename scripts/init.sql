-- ==============================================
-- WMS Lite - Inicialización de Base de Datos
-- =============================================

-- Extension para generar UUIDs (postgres < 12)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

--  Crear ENUM para tipo de movimiento
CREATE TYPE movement_type AS ENUM ('IN', 'OUT');

-- table users 
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_users_email ON users(email);


-- TABLE sessions
CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    refresh_token VARCHAR(255) UNIQUE NOT NULL,
    token_family VARCHAR(255) NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,

    device_name VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,

    parent_token_id INTEGER REFERENCES sessions(id) ON DELETE SET NULL
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX idx_sessions_token_family ON sessions(token_family);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Tabla de Productos
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

-- Tabla de Movimientos
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

-- Trigger para actualizar updated_at automáticamente
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
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Datos de prueba (opcional - comentar si no se necesita)
INSERT INTO products (sku, name, description, stock_quantity) VALUES
    ('LAPTOP-001', 'Laptop Dell XPS 15', 'Laptop profesional 15 pulgadas', 10),
    ('MOUSE-001', 'Mouse Logitech MX Master', 'Mouse ergonómico inalámbrico', 25),
    ('KEYBOARD-001', 'Teclado Mecánico Keychron K2', 'Teclado mecánico RGB', 0),
    ('MONITOR-001', 'Monitor Dell 27" 4K', 'Monitor UHD 27 pulgadas', 8)
ON CONFLICT (sku) DO NOTHING;

INSERT INTO users (email, password_hash, full_name, role) VALUES
('admin@wms.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Admin User', 'admin')
ON CONFLICT (email) DO NOTHING;

-- Mensaje de confirmación
DO $$
BEGIN
    RAISE NOTICE '✅ Base de datos inicializada correctamente';
END $$;