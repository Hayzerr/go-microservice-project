package clients

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

// ProductClient представляет клиент для взаимодействия с product-service
type ProductClient struct {
	baseURL string
	client  *http.Client
	// Для тестирования без реального сервиса
	mockMode bool
}

// Product представляет упрощенную модель продукта из product-service
type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

// NewProductClient создает новый экземпляр клиента для работы с product-service
func NewProductClient() *ProductClient {
	baseURL := os.Getenv("PRODUCT_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8082" // URL по умолчанию
	}

	// Включаем моковый режим для тестирования, если переменная окружения установлена
	mockMode := os.Getenv("MOCK_SERVICES") == "true"

	return &ProductClient{
		baseURL:  baseURL,
		client:   &http.Client{},
		mockMode: mockMode,
	}
}

// GetProductByID получает информацию о продукте по его ID
func (c *ProductClient) GetProductByID(productID int) (*Product, error) {
	// Если включен моковый режим, возвращаем моковые данные
	if c.mockMode {
		// Создаем моковый продукт с указанным ID
		return &Product{
			ID:          productID,
			Name:        fmt.Sprintf("Тестовый продукт %d", productID),
			Description: "Тестовое описание продукта",
			Price:       1000.0,
			Stock:       100,
		}, nil
	}

	url := fmt.Sprintf("%s/api/products/%d", c.baseURL, productID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка соединения с product-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("продукт не найден")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка получения продукта: код %d", resp.StatusCode)
	}

	var product Product
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	return &product, nil
}
