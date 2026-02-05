🎓 CONCEPTOS DE GO DOMINADOS
Básicos

✅ Punteros vs valores (*T vs T)
✅ Structs y composición
✅ Métodos con receivers (func (s*Struct) Method())
✅ Interfaces implícitas
✅ Error handling idiomático (if err != nil)
✅ Slices, maps, arrays
✅ Goroutines básicas
✅ Defer, panic, recover

Intermedios

✅ Struct tags (json:"field", db:"column", binding:"required")
✅ Context para timeouts y cancelación
✅ Closures (funciones que retornan funciones)
✅ Type aliases (type HandlerFunc func(*Context))
✅ Variadic functions (func foo(args ...string))
✅ Type assertions (value.(int), value, ok := x.(type))
✅ Embedded structs (composición)
✅ Punteros en structs para campos nullable (*string, *int)
✅ sql.NullString para campos DB nullable
✅ Early return pattern (validar y salir temprano)

Avanzados

✅ Dependency Injection manual
✅ Package-oriented design (capas: model → repo → service → handler)
✅ Middleware pattern con Gin
✅ c.Next() vs c.Abort() (control de flujo)
✅ c.Set() / c.Get() (contexto como "mochila")
✅ Route groups con middleware selectivo

🗄️ BASE DE DATOS (PostgreSQL)
Conceptos SQL Dominados

✅ Queries parametrizados ($1, $2, $3)
✅ Transacciones con BeginTxx, Commit, Rollback
✅ SELECT FOR UPDATE (locks pesimistas)
✅ Índices (simple y compuestos)
✅ ENUM types en PostgreSQL
✅ Triggers (actualización automática de updated_at)
✅ Foreign keys con ON DELETE RESTRICT/CASCADE
✅ CHECK constraints (stock_quantity >= 0)
✅ UNIQUE constraints
✅ RETURNING clause (obtener datos insertados)

sqlx (Librería)

✅ QueryRowContext + Scan
✅ QueryContext + rows.Next() + rows.Scan()
✅ ExecContext para INSERT/UPDATE/DELETE
✅ Connection pooling configurado
✅ Context timeouts en queries

🏗️ ARQUITECTURA Y PATRONES
Estructura del Proyecto
cmd/api/main.go              → Entrypoint, wiring
internal/
  platform/
    database.go              → Connection pool
    env.go                   → Config con structs
    logger.go                → Zerolog setup
    middleware.go            → HTTP logging
  product/
    model.go                 → DTOs y entities
    repository.go            → SQL queries
    service.go               → Business logic
    handler.go               → HTTP endpoints
  movement/
    (misma estructura)
  auth/
    model.go
    repository.go
    service.go
    handler.go
    middleware.go            → Auth & RBAC
scripts/init.sql             → Schema
compose.yaml
.env
Capas (Separation of Concerns)

✅ Model: Structs, DTOs, conversiones
✅ Repository: SQL puro, mapeo DB ↔ structs
✅ Service: Validaciones, lógica de negocio, transacciones
✅ Handler: Parsear HTTP, status codes, JSON

Patrones Aplicados

✅ Dependency Injection (pasar deps en constructores)
✅ Repository Pattern
✅ DTO Pattern (Entity vs Response)
✅ Middleware Pattern
✅ Builder Pattern (zerolog)

🔐 AUTENTICACIÓN Y SEGURIDAD
JWT (JSON Web Tokens)

✅ Anatomía del JWT (header.payload.signature)
✅ Claims estándar (exp, iat, iss)
✅ Claims personalizados (user_id, email, role)
✅ Firma con HMAC-SHA256
✅ Validación y parsing
✅ Access tokens (corta duración: 15 min)
✅ Refresh tokens (larga duración: 7 días)

Refresh Token Rotation (Enterprise)

✅ Token families (agrupar tokens rotados)
✅ parent_token_id (linaje de tokens)
✅ Detección de reutilización (seguridad anti-robo)
✅ Revocar familia completa cuando se detecta ataque
✅ Múltiples sesiones por usuario (web, mobile, etc.)
✅ Metadata de sesión (IP, User Agent, última actividad)

Password Hashing

✅ bcrypt con cost 10
✅ Salt automático en cada hash
✅ GenerateFromPassword y CompareHashAndPassword

RBAC (Role-Based Access Control)

✅ Roles: admin, user, readonly
✅ Middleware RequireAuth() (valida JWT)
✅ Middleware RequireRole(...roles) (valida permisos)
✅ 401 vs 403 (Unauthorized vs Forbidden)

📚 LIBRERÍAS Y HERRAMIENTAS
Gin (Web Framework)

✅ router.GET/POST/PUT/DELETE
✅ c.JSON(), c.Status()
✅ c.ShouldBindJSON() con validaciones
✅ c.Param(), c.Query(), c.GetHeader()
✅ c.ClientIP(), c.Request.UserAgent()
✅ c.Set(), c.Get() (contexto)
✅ c.Next(), c.Abort()
✅ Route groups
✅ Middleware con .Use()

Zerolog (Logging)

✅ Structured logging (JSON)
✅ Niveles: Debug, Info, Warn, Error, Fatal
✅ .Str(), .Int(), .Err(), .Dur()
✅ .Msg() (siempre al final)
✅ ConsoleWriter (dev) vs JSON (prod)
✅ Logger con contexto (.With().Str("module", "x"))

Otras Librerías

✅ sqlx - SQL con structs
✅ bcrypt - Password hashing
✅ jwt-go - JWT
✅ uuid - Generar IDs únicos
✅ godotenv - Variables de entorno
✅ cors (Gin middleware)

Herramientas

✅ Docker Compose (PostgreSQL)
✅ Air (hot reload)
✅ go mod (dependencias)
