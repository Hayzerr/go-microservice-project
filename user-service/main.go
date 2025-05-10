package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/example/microservices/pb"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedUserServiceServer
	db *sql.DB
}

// Структура пользователя для HTTP и JSON
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	grpcPort := getenv("GRPC_PORT", "50051")
	httpPort := getenv("HTTP_PORT", "8081")
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
	pb.RegisterUserServiceServer(g, &server{db: db})

	go func() {
		log.Printf("gRPC :%s", grpcPort)
		if err := g.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// REST (minimal)
	mux := http.NewServeMux()

	// Маршрут для создания пользователя
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var user User
			// Декодируем тело запроса
			err := json.NewDecoder(r.Body).Decode(&user)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Сохранение пользователя в базе данных
			_, err = db.Exec("INSERT INTO users (name, email) VALUES ($1, $2)", user.Name, user.Email)
			if err != nil {
				http.Error(w, "Error saving user to the database", http.StatusInternalServerError)
				return
			}

			// Ответ о успешном создании
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("User " + user.Name + " created"))
		} else if r.Method == http.MethodGet {
			// Обработка GET-запроса для получения всех пользователей
			rows, err := db.Query("SELECT id, name, email FROM users")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var users []User
			for rows.Next() {
				var user User
				if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				users = append(users, user)
			}

			// Устанавливаем тип контента как JSON
			w.Header().Set("Content-Type", "application/json")

			// Возвращаем список пользователей в формате JSON
			if err := json.NewEncoder(w).Encode(users); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	// Проверка здоровья сервиса
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// HTTP сервер
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

// Утилита для получения переменных окружения с дефолтными значениями
func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
