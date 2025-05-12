package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings" // Для извлечения ID из URL в handleUserByID, если используется стандартный ServeMux

	// ВАЖНО: Замените 'your_project_module' на имя вашего модуля из go.mod
	"your_project_module/internal/user/models" // Может понадобиться для ответа
	"your_project_module/internal/user/usecase"

	// Для более удобной маршрутизации и извлечения параметров URL можно использовать
	// "github.com/go-chi/chi/v5" или "github.com/gorilla/mux".
	// Здесь для простоты используется стандартный http.ServeMux,
	// что делает извлечение параметров пути (как {id}) менее удобным.
	// Если вы используете chi, например:
	// import "github.com/go-chi/chi/v5"
)

// UserHTTPHandler обрабатывает HTTP запросы, связанные с пользователями.
type UserHTTPHandler struct {
	userUsecase usecase.UserUsecase
}

// NewUserHTTPHandler создает новый экземпляр UserHTTPHandler.
func NewUserHTTPHandler(uc usecase.UserUsecase) *UserHTTPHandler {
	return &UserHTTPHandler{userUsecase: uc}
}

// RegisterRoutes регистрирует HTTP маршруты для обработчика пользователей.
// Принимает *http.ServeMux, но можно адаптировать для других роутеров.
func (h *UserHTTPHandler) RegisterRoutes(router *http.ServeMux) {
	// Для /api/users: POST для создания
	router.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.createUser(w, r)
		} else {
			http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		}
	})

	// Для /api/users/{id}: GET для получения, PUT для обновления
	// Обратите внимание: стандартный ServeMux не поддерживает параметры пути типа {id} напрямую.
	// Мы будем извлекать ID из r.URL.Path.
	// Более продвинутые роутеры (chi, gorilla/mux) делают это элегантнее.
	router.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем ID из пути. Пример: /api/users/some-uuid-string
		// Этот способ извлечения ID подходит для путей, заканчивающихся на ID.
		// Если есть еще что-то после ID, потребуется более сложный парсинг.
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		var userID string
		if len(pathParts) == 3 && pathParts[0] == "api" && pathParts[1] == "users" {
			userID = pathParts[2]
		}

		if userID == "" {
			http.Error(w, "ID пользователя не указан в пути или путь некорректен", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.getUserByID(w, r, userID)
		case http.MethodPut:
			h.updateUser(w, r, userID)
		// case http.MethodDelete:
		// 	h.deleteUser(w, r, userID) // TODO: Реализовать
		default:
			http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		}
	})
	// router.HandleFunc("/api/auth/login", h.handleLogin) // TODO: Реализовать
}

// createUser обрабатывает запрос на создание нового пользователя.
func (h *UserHTTPHandler) createUser(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Некорректное тело запроса: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if requestBody.Username == "" || requestBody.Email == "" || requestBody.Password == "" {
		http.Error(w, "Имя пользователя, email и пароль не могут быть пустыми", http.StatusBadRequest)
		return
	}

	user, err := h.userUsecase.RegisterUser(r.Context(), requestBody.Username, requestBody.Email, requestBody.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrEmailExists) {
			http.Error(w, "Пользователь с таким email уже существует", http.StatusConflict)
			return
		}
		if errors.Is(err, usecase.ErrPasswordTooShort) {
			http.Error(w, "Пароль слишком короткий", http.StatusBadRequest)
			return
		}
		http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Убираем пароль из ответа (хотя usecase уже должен это делать)
	user.Password = ""
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// getUserByID обрабатывает запрос на получение пользователя по ID.
func (h *UserHTTPHandler) getUserByID(w http.ResponseWriter, r *http.Request, userID string) {
	user, err := h.userUsecase.FindUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}
		http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Убираем пароль из ответа (usecase уже должен это делать, но для надежности)
	user.Password = ""
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// updateUser обрабатывает запрос на обновление пользователя.
func (h *UserHTTPHandler) updateUser(w http.ResponseWriter, r *http.Request, userID string) {
	var requestBody struct {
		Username *string `json:"username,omitempty"` // omitempty, чтобы null не превращался в пустую строку при десериализации
		Email    *string `json:"email,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Некорректное тело запроса: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	updateInput := usecase.UpdateUserInput{
		Username: requestBody.Username,
		Email:    requestBody.Email,
	}

	// Проверка, есть ли вообще что обновлять
	if updateInput.Username == nil && updateInput.Email == nil {
		http.Error(w, "Нет данных для обновления", http.StatusBadRequest)
		return
	}

	updatedUser, err := h.userUsecase.UpdateUser(r.Context(), userID, updateInput)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			http.Error(w, "Пользователь для обновления не найден", http.StatusNotFound)
		case errors.Is(err, usecase.ErrEmailExists):
			http.Error(w, "Новый email уже используется другим пользователем", http.StatusConflict)
		default:
			http.Error(w, "Внутренняя ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Убираем пароль из ответа (usecase уже должен это делать)
	updatedUser.Password = ""
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedUser)
}

// TODO: Реализуйте другие HTTP обработчики (login, delete, list).
// Для login (аутентификации) вам, вероятно, понадобится возвращать JWT или другой токен сессии.
// Для deleteUser: метод DELETE /api/users/{id}
// Для listUsers: метод GET /api/users (потребует доработки RegisterRoutes и handleUsers)
