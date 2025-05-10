FROM golang:1.22-alpine

WORKDIR /app

COPY product-service/go.mod product-service/go.sum ./
COPY pb ./pb

RUN go mod download

COPY product-service/. .

RUN go build -o service .
CMD ["./service"]
