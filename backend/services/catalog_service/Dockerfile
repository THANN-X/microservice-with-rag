FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY backend/pkg ./pkg
COPY backend/services/catalog_service ./services/catalog_service

WORKDIR /app/services/catalog_service

RUN go mod download

RUN go build -o main ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/services/catalog_service/main .
EXPOSE 3005
CMD [ "./main" ]
