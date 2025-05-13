package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Драйвер базы данных PostgreSQL
	_ "github.com/lib/pq"

	// gRPC
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// ВАЖНО: Замените пути на актуальные для вашего проекта
	// Путь к модулю с protobuf определениями (из pb/go.mod)
	pb "github.com/Hayzerr/go-microservice-project/pb" // Пример

	// Пути к внутренним пакетам product-service.
	// Замените "github.com/Hayzerr/go-microservice-project/product-service"
	// на актуальное имя модуля из product-service/go.mod, если оно другое.
	grpcProductDelivery "github.com/Hayzerr/go-microservice-project/product-service/internal/product/delivery/grpc"
	httpProductDelivery "github.com/Hayzerr/go-microservice-project/product-service/internal/product/delivery/http"
	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/repository"
	"github.com/Hayzerr/go-microservice-project/product-service/internal/product/usecase"
)

// getenv получает значение переменной окружения или возвращает значение по умолчанию.
func getenv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	// Конфигурация портов и DSN для product-service
	// Используйте другие порты, отличные от user-service
	grpcPort := getenv("PRODUCT_GRPC_PORT", "50052")
	httpPort := getenv("PRODUCT_HTTP_PORT", "8082")
	// Рекомендуется использовать отдельную БД или схему для product-service
	dsn := getenv("PRODUCT_DB_DSN", "host=postgres-product port=5432 user=postgres password=postgres dbname=product_service_db sslmode=disable")

	log.Println("Запуск product-service...")

	// 1. Инициализация соединения с базой данных
	log.Println("Подключение к базе данных (product_service_db)...")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка проверки соединения с базой данных: %v", err)
	}
	log.Println("Успешное подключение к базе данных (product_service_db).")

	// 2. Создание экземпляра репозитория
	productRepo := repository.NewProductRepository(db)
	log.Println("Репозиторий продуктов инициализирован.")

	// 3. Создание экземпляра бизнес-логики (usecase)
	productUsecase := usecase.NewProductUsecase(productRepo)
	log.Println("Бизнес-логика продуктов инициализирована.")

	// 4. Создание экземпляра gRPC обработчика
	productGRPCHandler := grpcProductDelivery.NewProductGRPCHandler(productUsecase)
	log.Println("gRPC обработчик продуктов инициализирован.")

	// 5. Создание экземпляра HTTP обработчика
	productHTTPHandler := httpProductDelivery.NewProductHTTPHandler(productUsecase, productRepo)
	log.Println("HTTP обработчик продуктов инициализирован.")

	var gRPCServer *grpc.Server
	var httpServer *http.Server

	// Запуск gRPC сервера
	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("Ошибка прослушивания gRPC порта %s для product-service: %v", grpcPort, err)
		}
		gRPCServer = grpc.NewServer()
		pb.RegisterProductServiceServer(gRPCServer, productGRPCHandler) // Регистрация ProductService
		reflection.Register(gRPCServer)

		log.Printf("Product-service gRPC сервер запущен на порту: %s", grpcPort)
		if err := gRPCServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("Ошибка запуска gRPC сервера для product-service: %v", err)
		} else if errors.Is(err, grpc.ErrServerStopped) {
			log.Println("Product-service gRPC сервер штатно остановлен.")
		}
	}()

	// Запуск HTTP сервера
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { // Общий health check
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		// Регистрация маршрутов для product HTTP handler
		productHTTPHandler.RegisterRoutes(mux)

		httpServer = &http.Server{
			Addr:    ":" + httpPort,
			Handler: mux,
		}

		log.Printf("Product-service HTTP сервер запущен на порту: %s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка запуска HTTP сервера для product-service: %v", err)
		} else if errors.Is(err, http.ErrServerClosed) {
			log.Println("Product-service HTTP сервер штатно остановлен.")
		}
	}()

	// Ожидание сигнала для корректного завершения работы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Product-service: получен сигнал %v, начинаю корректное выключение...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if httpServer != nil {
		log.Println("Product-service: остановка HTTP сервера...")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Product-service: ошибка корректного завершения HTTP сервера: %v", err)
		} else {
			log.Println("Product-service: HTTP сервер успешно остановлен.")
		}
	}

	if gRPCServer != nil {
		log.Println("Product-service: остановка gRPC сервера...")
		gRPCServer.GracefulStop()
		log.Println("Product-service: gRPC сервер успешно остановлен.")
	}
	log.Println("Сервис product-service полностью остановлен.")
}
