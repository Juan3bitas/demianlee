# ---------- Etapa 1: compilación ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copiamos primero el go.mod para aprovechar el cache de capas de Docker
COPY go.mod ./
RUN go mod download 2>/dev/null || true

COPY . .

# Compilamos un binario estático (sin dependencias de C) para que corra
# en una imagen final mínima, sin importar la infraestructura destino.
RUN CGO_ENABLED=0 GOOS=linux go build -o demian-lee .

# ---------- Etapa 2: imagen final ----------
FROM alpine:3.20

# Certificados TLS por si en el futuro se agregan llamadas externas
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/demian-lee .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/data ./data

EXPOSE 8080

ENV PORT=8080

CMD ["./demian-lee"]
