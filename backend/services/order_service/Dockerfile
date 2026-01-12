FROM golang:1.25.4-alpine As builder

WORKDIR /app

COPY backend/pkg ./pkg

COPY backend/services/order_service ./services/order_service

WORKDIR /app/services/order_service

RUN go mod download

RUN go build -o main ./cmd


FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/services/order_service/main .
EXPOSE 3003
CMD [ "./main" ]