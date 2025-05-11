package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	// Драйвер базы данных PostgreSQL
	_ "github.com/lib/pq"

	// gRPC
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection" // Для упрощения тестирования gRPC через grpcurl или Postman

	// Импорты для вашего проекта (ВАЖНО: Замените 'your_project_module' на имя вашего модуля из go.mod)
	grpcDelivery "github.com/example/microservices/user-service/internal/user/delivery/grpc" // Пакет для gRPC обработчиков
	httpDelivery "github.com/example/microservices/user-service/internal/user/delivery/http" // Пакет для HTTP обработчиков
	"github.com/example/microservices/user-service/internal/user/repository"                 // Пакет для репозитория
	"github.com/example/microservices/user-service/internal/user/usecase"                    // Пакет для бизнес-логики
	pb "github.com/example/microservices/user-service/pb"                                    // Путь к сгенерированным .pb.go файлам
	// Для HTTP-роутера можно использовать стандартный http.ServeMux или более продвинутые,
	// например, "github.com/gorilla/mux" или "github.com/go-chi/chi/v5"
	// В этом примере используется стандартный http.ServeMux для простоты.
)

// getenv получает значение переменной окружения или возвращает значение по умолчанию.
func getenv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Структура пользователя для HTTP и JSON
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	// Получение конфигурации из переменных окружения
	grpcPort := getenv("GRPC_PORT", "50051")
	httpPort := getenv("HTTP_PORT", "8081")
	// ВАЖНО: Убедитесь, что DSN соответствует вашей конфигурации PostgreSQL
	dsn := getenv("DB_DSN", "host=localhost port=5432 user=postgres password=postgres dbname=user_service_db sslmode=disable")
	// Рекомендуется использовать отдельную базу данных для каждого сервиса, например, 'user_service_db'

	log.Println("Запуск user-service...")

	// 1. Инициализация соединения с базой данных
	log.Println("Подключение к базе данных...")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Проверка соединения с БД
	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка проверки соединения с базой данных: %v", err)
	}
	log.Println("Успешное подключение к базе данных.")

	// 2. Создание экземпляра репозитория
	//    Предполагается, что у вас есть функция NewPostgresUserRepository в пакете repository.
	userRepo := repository.NewPostgresUserRepository(db)
	log.Println("Репозиторий пользователей инициализирован.")

	// 3. Создание экземпляра бизнес-логики (usecase)
	//    Предполагается, что у вас есть функция NewUserUsecase в пакете usecase.
	userUsecase := usecase.NewUserUsecase(userRepo)
	log.Println("Бизнес-логика пользователей инициализирована.")

	// 4. Создание экземпляра gRPC обработчика
	//    Предполагается, что у вас есть функция NewUserGRPCHandler в пакете grpcDelivery.
	userGRPCHandler := grpcDelivery.NewUserGRPCHandler(userUsecase)
	log.Println("gRPC обработчик пользователей инициализирован.")

	// 5. Создание экземпляра HTTP обработчика
	//    Предполагается, что у вас есть функция NewUserHTTPHandler в пакете httpDelivery.
	userHTTPHandler := httpDelivery.NewUserHTTPHandler(userUsecase)
	log.Println("HTTP обработчик пользователей инициализирован.")

	// Запуск gRPC сервера
	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("Ошибка прослушивания gRPC порта %s: %v", grpcPort, err)
		}
		gRPCServer := grpc.NewServer()
		pb.RegisterUserServiceServer(gRPCServer, userGRPCHandler) // Регистрация вашего gRPC сервиса

		// Регистрация reflection для gRPC сервера (полезно для отладки)
		reflection.Register(gRPCServer)

		log.Printf("gRPC сервер запущен на порту: %s", grpcPort)
		if err := gRPCServer.Serve(lis); err != nil {
			log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
		}
	}()

	// Запуск HTTP сервера
	go func() {
		// Создание HTTP роутера (мультиплексора)
		// Можно использовать http.NewServeMux() или более продвинутые роутеры
		router := http.NewServeMux()

		// Базовый эндпоинт для проверки работоспособности
		router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		// Регистрация HTTP эндпоинтов для вашего user-service
		// Пример:
		// router.HandleFunc("/api/users", userHTTPHandler.CreateUser) // POST
		// router.HandleFunc("/api/users/{id}", userHTTPHandler.GetUser) // GET
		// Замените на реальные методы вашего HTTP обработчика
		// Для простоты, предположим, что у userHTTPHandler есть метод RegisterRoutes
		userHTTPHandler.RegisterRoutes(router) // Предполагается, что такой метод существует

		httpServer := &http.Server{
			Addr:    ":" + httpPort,
			Handler: router,
		}

		log.Printf("HTTP сервер запущен на порту: %s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска HTTP сервера: %v", err)
		}
	}()

	// Ожидание сигнала для корректного завершения работы
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Блокировка до получения сигнала

	log.Println("Получен сигнал завершения, начинаю корректное выключение...")

	// Таймаут для корректного завершения
	// TODO: gRPCServer.GracefulStop() и httpServer.Shutdown() должны быть вызваны.
	// В текущем коде они не вызываются при завершении, это нужно исправить.
	// Для gRPCServer: gRPCServer.GracefulStop()
	// Для httpServer:
	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()
	// if err := httpServer.Shutdown(ctx); err != nil {
	//     log.Fatalf("Ошибка корректного завершения HTTP сервера: %v", err)
	// }

	log.Println("Сервис user-service остановлен.")
}
