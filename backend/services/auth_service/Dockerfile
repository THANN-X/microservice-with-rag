FROM golang:1.25.4-alpine As builder

WORKDIR /app

COPY backend/pkg ./pkg

COPY backend/services/auth_service ./services/auth_service

WORKDIR /app/services/auth_service

RUN go mod download

RUN go build -o main ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/services/auth_service/main .
EXPOSE 3001
CMD [ "./main" ]