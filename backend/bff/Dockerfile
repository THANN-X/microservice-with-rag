FROM golang:1.25.4-alpine As builder

WORKDIR /app

COPY backend/pkg ./pkg

COPY backend/bff ./bff

WORKDIR /app/bff

RUN go mod download

RUN go build -o main ./cmd

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/bff/main .

EXPOSE 8080

CMD [ "./main" ]