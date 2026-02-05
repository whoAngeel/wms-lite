# 📝 Roadmap & TODOs

## 🚀 Fase 1: Verificación y Estabilización (Prioridad Alta)

*Objetivo: Asegurar que el núcleo actual funciona correctamente y es robusto.*

### Análisis de Flujo Auth & Seguridad

- [ ] Registrar usuario
- [ ] Login (recibir access + refresh token)
- [ ] Usar access token en endpoints protegidos
- [ ] Refresh cuando expira el token
- [ ] Logout (Revocación)
- [ ] Probar RBAC (Validar pemisos de diferentes roles)
- [ ] Probar detección de reutilización de tokens

### Funcionalidades Core Faltantes

- [ ] **Soft Delete**: Implementar borrado lógico en lugar de físico.
- [ ] **Búsqueda Avanzada**: Implementar filtros de búsqueda más potentes.
- [ ] **Exportar Reportes**: Generación de CSV para datos clave.

---

## 🧪 Fase 2: Estrategia de Testing

*Objetivo: Elevar la calidad del código y prevenir bugs futuros.*

### B1. Unit Tests (~2 horas)

- [ ] Tests de Service Layer
- [ ] Mocking de Repository Layer
- [ ] Implementar Table-driven tests (patrón standard de Go)
- [ ] Meta: Code Coverage > 80%

### B2. Integration Tests (~1.5 horas)

- [ ] Tests de endpoints completos (HTTP Requests reales)
- [ ] Test de flujo de autenticación completo (End-to-End)
- [ ] Tests de concurrencia / Race conditions

---

## 🛡️ Fase 3: Features Avanzadas

*Objetivo: Mejorar la seguridad y observabilidad del sistema.*

### C1. Gestión de Sesiones Avanzada (~1 hora)

- [ ] **Refresh Token Rotation Mejorado**: Detectar logins concurrentes sospechosos.
- [ ] Email de alerta ante detección de reutilización de tokens.
- [ ] Dashboard de sesiones activas.

### C2. Rate Limiting (~45 min)

- [ ] Limitar requests por IP para prevenir abuso.
- [ ] Middleware con Redis o In-Memory map.
- [ ] Configurar límites diferenciados por endpoint.

### C3. Audit Logging (~1 hora)

- [ ] Crear tabla `audit_logs`.
- [ ] Registrar operaciones críticas (Create/Update/Delete).
- [ ] Trazabilidad completa: Quién, Qué, Cuándo (y Snapshot del cambio).

---

## 💻 Fase 4: Frontend (Full Stack)

*Objetivo: Crear una interfaz visual para el sistema.*

### D1. SPA con React/Vue (~3-4 horas)

- [ ] Formularios de Login y Registro.
- [ ] Dashboard principal con lista de productos.
- [ ] Interfaces para Crear/Editar productos.
- [ ] Vista de sesiones activas.
- [ ] Manejo automático de Refresh Tokens (Interceptors).

---

## 🚢 Fase 5: DevOps & Deployment

*Objetivo: Automatizar la entrega y optimizar recursos.*

### E1. Docker Optimization (~30 min)

- [ ] Implementar Multi-Stage Build.
- [ ] Reducir imagen de producción a < 20MB (Scratch/Alpine).

### E2. CI/CD Pipeline (~1 hora)

- [ ] Configurar GitHub Actions.
- [ ] Ejecución automática de Tests en PRs.
- [ ] Deploy automático a Railway/Render/Fly.io.
