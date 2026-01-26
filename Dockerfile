# Etapa 1: Builder - Compilar la aplicación
FROM golang:1.25-alpine AS builder

# Instalar dependencias del sistema
RUN apk add --no-cache git

# Establecer directorio de trabajo
WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar la aplicación
# CGO_ENABLED=0: binario estático (no depende de librerías C)
# -ldflags="-w -s": reduce tamaño del binario (strip debug info)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api

# Etapa 2: Runtime - Imagen mínima para ejecutar
FROM alpine:latest

# Instalar certificados SSL (para HTTPS si es necesario)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copiar binario compilado desde builder
COPY --from=builder /app/bin/api .

# Exponer puerto
EXPOSE 4002

# Comando de inicio
CMD ["./api"]