FROM golang:1.22-alpine

WORKDIR /app

COPY order-service/go.mod order-service/go.sum ./
COPY pb ./pb

RUN go mod download

COPY order-service/. .

RUN go build -o service .
CMD ["./service"]
