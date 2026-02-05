# LOGGING


// ✅ SÍ loggear:
- Creaciones exitosas (Info)
- Actualizaciones exitosas (Info)
- Eliminaciones exitosas (Info)
- Errores del servidor (Error)
- Validaciones fallidas importantes (Warn)

// ❌ NO loggear:
- GET exitosos (ya lo loggea el middleware HTTP)
- 404 normales (ya lo loggea el middleware)
- Validaciones triviales