FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY backend/pkg ./pkg

COPY backend/services/cart_service ./services/cart_service

WORKDIR /app/services/cart_service

RUN go mod download

RUN go build -o main ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/services/cart_service/main .
EXPOSE 3004
CMD [ "./main" ]
