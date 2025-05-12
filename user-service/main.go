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

	_ "github.com/lib/pq"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/Hayzerr/go-microservice-project/pb"
	grpcDelivery "github.com/Hayzerr/go-microservice-project/user-service/internal/user/delivery/grpc"
	httpDelivery "github.com/Hayzerr/go-microservice-project/user-service/internal/user/delivery/http"
	"github.com/Hayzerr/go-microservice-project/user-service/internal/user/repository"
	"github.com/Hayzerr/go-microservice-project/user-service/internal/user/usecase"
)

func getenv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	grpcPort := getenv("GRPC_PORT", "50051")
	httpPort := getenv("HTTP_PORT", "8081")
	dsn := getenv("DB_DSN", "host=localhost port=5432 user=postgres password=postgres dbname=user_service_db sslmode=disable")

	log.Println("Запуск user-service...")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка проверки соединения с базой данных: %v", err)
	}
	log.Println("Успешное подключение к базе данных.")

	userRepo := repository.NewPostgresUserRepository(db)
	hasher := usecase.NewBcryptPasswordHasher(0)
	jwtManager := usecase.NewJWTManager("supersecretkey", 24*time.Hour)
	userUsecase := usecase.NewUserUsecase(userRepo, hasher)

	// gRPC handler
	userGRPCHandler := grpcDelivery.NewUserGRPCHandler(userUsecase)

	// HTTP handler
	userHTTPHandler := httpDelivery.NewUserHTTPHandler(userUsecase, jwtManager)

	// gRPC сервер
	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("Ошибка прослушивания gRPC порта %s: %v", grpcPort, err)
		}
		gRPCServer := grpc.NewServer()
		pb.RegisterUserServiceServer(gRPCServer, userGRPCHandler)
		reflection.Register(gRPCServer)
		log.Printf("gRPC сервер запущен на порту: %s", grpcPort)
		if err := gRPCServer.Serve(lis); err != nil {
			log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
		}
	}()

	// HTTP сервер
	httpServer := &http.Server{
		Addr: ":" + httpPort,
	}

	go func() {
		router := http.NewServeMux()
		router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		userHTTPHandler.RegisterRoutes(router)
		httpServer.Handler = router

		log.Printf("HTTP сервер запущен на порту: %s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска HTTP сервера: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Получен сигнал завершения, начинаю корректное выключение...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка корректного завершения HTTP сервера: %v", err)
	}

	log.Println("Сервис user-service остановлен.")
}
