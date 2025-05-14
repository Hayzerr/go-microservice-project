package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hayzerr/go-microservice-project/order-service/internal/clients"
	orderHttp "github.com/Hayzerr/go-microservice-project/order-service/internal/order/delivery/http"
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/repository"
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/usecase"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"

	pb "github.com/Hayzerr/go-microservice-project/pb"
)

type server struct {
	pb.UnimplementedOrderServiceServer
}

// checkServiceAvailability проверяет доступность сервиса по указанному URL
func checkServiceAvailability(url string) bool {
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(url + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func main() {
	// Получаем конфигурацию из переменных окружения
	grpcPort := getenv("GRPC_PORT", "50053")
	httpPort := getenv("HTTP_PORT", "8083")

	// Проверяем доступность других сервисов
	userServiceURL := getenv("USER_SERVICE_URL", "http://localhost:8081")
	productServiceURL := getenv("PRODUCT_SERVICE_URL", "http://localhost:8082")

	userServiceAvailable := checkServiceAvailability(userServiceURL)
	productServiceAvailable := checkServiceAvailability(productServiceURL)

	// Если сервисы недоступны, включаем моковый режим
	if !userServiceAvailable || !productServiceAvailable {
		log.Println("Внимание: некоторые сервисы недоступны, включен моковый режим")
		if !userServiceAvailable {
			log.Println("User-service недоступен:", userServiceURL)
		}
		if !productServiceAvailable {
			log.Println("Product-service недоступен:", productServiceURL)
		}
		os.Setenv("MOCK_SERVICES", "true")
	}

	// Инициализируем клиенты для других сервисов
	userClient := clients.NewUserClient()
	productClient := clients.NewProductClient()

	// Инициализируем репозиторий (in-memory)
	orderRepo := repository.NewMemoryRepository()

	// Инициализируем usecase
	orderUseCase := usecase.NewOrderUseCase(orderRepo, userClient, productClient)

	// Инициализируем HTTP-обработчики
	orderHandler := orderHttp.NewHandler(orderUseCase)

	// gRPC сервер
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	g := grpc.NewServer()
	pb.RegisterOrderServiceServer(g, &server{})

	go func() {
		log.Printf("gRPC server listening on :%s", grpcPort)
		if err := g.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// HTTP сервер
	router := mux.NewRouter()

	// Регистрируем маршруты
	orderHandler.RegisterRoutes(router)

	// Добавляем маршрут для проверки работоспособности
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: router,
	}

	go func() {
		log.Printf("HTTP server listening on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	// Обработка сигналов для корректного завершения
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Println("shutting down servers...")

	// Корректное завершение HTTP сервера
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Корректное завершение gRPC сервера
	g.GracefulStop()

	log.Println("servers stopped")
}

func getenv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
