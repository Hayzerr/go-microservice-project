package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/Hayzerr/go-microservice-project/pb"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedOrderServiceServer
	db *sql.DB
}

func main() {
	grpcPort := getenv("GRPC_PORT", "50053")
	httpPort := getenv("HTTP_PORT", "8083")
	dsn := getenv("DB_DSN", "host=localhost user=postgres password=postgres dbname=postgres sslmode=disable")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	// gRPC
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	g := grpc.NewServer()
	pb.RegisterOrderServiceServer(g, &server{db: db})

	go func() {
		log.Printf("gRPC :%s", grpcPort)
		if err := g.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// REST (minimal)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: mux,
	}
	go func() {
		log.Printf("HTTP :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http serve: %v", err)
		}
	}()

	// Graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	log.Println("shutdown...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	g.GracefulStop()
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
