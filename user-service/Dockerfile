FROM golang:1.22-alpine

WORKDIR /app

COPY user-service/go.mod user-service/go.sum ./
COPY pb ./pb

RUN go mod download

COPY user-service/. .

RUN go build -o service .
CMD ["./service"]
