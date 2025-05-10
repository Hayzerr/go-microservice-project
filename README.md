# Go Microservices Demo

This repository contains a **three‑microservice** demo system written in Go:

| Service | Domain | gRPC Port | REST Port |
|---------|--------|-----------|-----------|
| user‑service    | User management (auth/profile) | 50051 | 8081 |
| product‑service | Catalog & inventory            | 50052 | 8082 |
| order‑service   | Order processing               | 50053 | 8083 |

## Tech stack
* Go 1.22
* gRPC (+ protobuf)
* Standard library HTTP for REST
* PostgreSQL (via `database/sql` + `pq`)
* Docker & docker‑compose

## Quick start
```bash
# 1. build & start everything
docker compose up --build

# 2. check health
curl localhost:8081/healthz
```

Proto definitions live in **/proto** – run the helper to regenerate stubs:

```bash
make proto
```

Each service follows the same layout:

```
service-name/
  main.go            # bootstraps gRPC & REST
  handler/           # gRPC + REST handlers
  db/                # SQL + repository pattern
  Dockerfile
```

> **NOTE**  
> CRUD methods are stubbed with TODOs so the skeleton compiles; fill in
> business logic, database queries, and tests as you iterate.

## Tests & benchmarking
Run unit tests:
```bash
go test ./...
```

Benchmark an example function:
```bash
go test -run=NONE -bench=. ./order-service/...
```
