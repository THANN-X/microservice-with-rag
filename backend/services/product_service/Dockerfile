FROM golang:1.25.4-alpine As builder

WORKDIR /app

COPY backend/pkg ./pkg

COPY backend/services/product_service ./services/product_service

WORKDIR /app/services/product_service

RUN go mod download

RUN go build -o main ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/services/product_service/main .
EXPOSE 3002
CMD [ "./main" ]