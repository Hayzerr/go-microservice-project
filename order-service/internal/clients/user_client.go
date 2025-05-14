package clients

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

// UserClient представляет клиент для взаимодействия с user-service
type UserClient struct {
	baseURL string
	client  *http.Client
	// Для тестирования без реального сервиса
	mockMode bool
}

// User представляет упрощенную модель пользователя из user-service
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// NewUserClient создает новый экземпляр клиента для работы с user-service
func NewUserClient() *UserClient {
	baseURL := os.Getenv("USER_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8081" // URL по умолчанию
	}

	// Включаем моковый режим для тестирования, если переменная окружения установлена
	mockMode := os.Getenv("MOCK_SERVICES") == "true"

	return &UserClient{
		baseURL:  baseURL,
		client:   &http.Client{},
		mockMode: mockMode,
	}
}

// GetUserByID получает информацию о пользователе по его ID
func (c *UserClient) GetUserByID(userID string) (*User, error) {
	// Если включен моковый режим, возвращаем моковые данные
	if c.mockMode {
		return &User{
			ID:       userID,
			Username: "test_user",
			Email:    "test@example.com",
		}, nil
	}

	url := fmt.Sprintf("%s/api/users/%s", c.baseURL, userID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка соединения с user-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("пользователь не найден")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка получения пользователя: код %d", resp.StatusCode)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return &user, nil
}
