FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY backend/pkg ./pkg
COPY backend/services/order_history_service ./services/order_history_service

WORKDIR /app/services/order_history_service

RUN go mod download

RUN go build -o main ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/services/order_history_service/main .
EXPOSE 3006
CMD [ "./main" ]
