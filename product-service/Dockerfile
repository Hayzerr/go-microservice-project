FROM golang:1.22-alpine

WORKDIR /app

# Set up proper go module hierarchy for pb
COPY github.com/Hayzerr/go-microservice-project/pb /app/github.com/Hayzerr/go-microservice-project/pb
COPY product-service ./product-service
WORKDIR /app/product-service

RUN go mod edit -replace=github.com/Hayzerr/go-microservice-project/pb=../github.com/Hayzerr/go-microservice-project/pb
RUN go mod tidy
RUN go build -o service .

CMD ["./service"]
