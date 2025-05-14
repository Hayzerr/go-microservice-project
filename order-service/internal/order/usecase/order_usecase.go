package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt" // Для форматирования ошибок

	// ВАЖНО: Замените 'your_order_module_path' и 'your_pb_module_path'
	// на актуальные пути к вашим модулям.
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/repository"
	pb "github.com/Hayzerr/go-microservice-project/pb" // Сгенерированные proto-файлы для User и Product сервисов

	"google.golang.org/grpc"
	// "google.golang.org/grpc/status" // Для более детальной обработки gRPC ошибок
	// "google.golang.org/grpc/codes"
)

var (
	ErrOrderNotFound         = errors.New("заказ не найден")
	ErrInvalidOrderInput     = errors.New("некорректные входные данные для заказа")
	ErrUserNotFoundForOrder  = errors.New("пользователь для заказа не найден")
	ErrProductNotFoundForOrder = errors.New("один или несколько продуктов для заказа не найдены")
	ErrInsufficientStock     = errors.New("недостаточно товара на складе")
	ErrCreateOrderFailed     = errors.New("не удалось создать заказ")
	// Добавьте другие специфичные ошибки
)

// CreateOrderItemInput определяет входные данные для одной позиции при создании заказа.
type CreateOrderItemInput struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// CreateOrderInput определяет входные данные для создания нового заказа.
type CreateOrderInput struct {
	UserID string                 `json:"user_id"`
	Items  []CreateOrderItemInput `json:"items"`
}

// UserServiceClient определяет интерфейс для клиента UserService.
// Это позволяет мокировать клиент в тестах.
type UserServiceClient interface {
	GetUser(ctx context.Context, in *pb.GetUserRequest, opts ...grpc.CallOption) (*pb.GetUserResponse, error)
	// Добавьте другие методы UserService, если они понадобятся
}

// ProductServiceClient определяет интерфейс для клиента ProductService.
type ProductServiceClient interface {
	GetProduct(ctx context.Context, in *pb.GetProductRequest, opts ...grpc.CallOption) (*pb.GetProductResponse, error)
	// TODO: Нужен метод для уменьшения стока в ProductService, например, DecreaseStock
	// UpdateProduct(ctx context.Context, in *pb.UpdateProductRequest, opts ...grpc.CallOption) (*pb.UpdateProductResponse, error)
}

// OrderUsecase определяет интерфейс для бизнес-логики заказов.
type OrderUsecase interface {
	CreateOrder(ctx context.Context, input CreateOrderInput) (*models.Order, error)
	GetOrderByID(ctx context.Context, orderID string) (*models.Order, error)
	ListOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) (*models.Order, error)
}

type orderUsecase struct {
	orderRepo   repository.OrderRepository
	userClient  UserServiceClient    // gRPC клиент к UserService
	productClient ProductServiceClient // gRPC клиент к ProductService
}

// NewOrderUsecase создает новый экземпляр orderUsecase.
func NewOrderUsecase(
	orderRepo repository.OrderRepository,
	userClient UserServiceClient,
	productClient ProductServiceClient,
) OrderUsecase {
	return &orderUsecase{
		orderRepo:   orderRepo,
		userClient:  userClient,
		productClient: productClient,
	}
}

// CreateOrder создает новый заказ.
func (uc *orderUsecase) CreateOrder(ctx context.Context, input CreateOrderInput) (*models.Order, error) {
	// 1. Валидация входных данных
	if input.UserID == "" {
		return nil, fmt.Errorf("%w: UserID не может быть пустым", ErrInvalidOrderInput)
	}
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("%w: заказ должен содержать хотя бы одну позицию", ErrInvalidOrderInput)
	}

	// 2. Проверка существования пользователя через UserService
	_, err := uc.userClient.GetUser(ctx, &pb.GetUserRequest{Id: input.UserID})
	if err != nil {
		// TODO: Более детальная обработка gRPC ошибок (например, codes.NotFound)
		// st, ok := status.FromError(err)
		// if ok && st.Code() == codes.NotFound {
		//  return nil, ErrUserNotFoundForOrder
		// }
		return nil, fmt.Errorf("ошибка при проверке пользователя '%s': %w; %w", input.UserID, err, ErrUserNotFoundForOrder)
	}

	// 3. Обработка позиций заказа и расчет общей стоимости
	var orderItems []models.OrderItem
	var totalPrice float64

	for _, itemInput := range input.Items {
		if itemInput.Quantity <= 0 {
			return nil, fmt.Errorf("%w: количество для продукта ID '%s' должно быть больше нуля", ErrInvalidOrderInput, itemInput.ProductID)
		}

		// 3.1 Получение информации о продукте из ProductService
		productResp, err := uc.productClient.GetProduct(ctx, &pb.GetProductRequest{Id: itemInput.ProductID})
		if err != nil {
			// TODO: Более детальная обработка gRPC ошибок
			return nil, fmt.Errorf("ошибка при получении продукта ID '%s': %w; %w", itemInput.ProductID, err, ErrProductNotFoundForOrder)
		}
		productPb := productResp.GetProduct()
		if productPb == nil {
			return nil, fmt.Errorf("%w: продукт ID '%s' не найден", ErrProductNotFoundForOrder, itemInput.ProductID)
		}

		// 3.2 Проверка наличия на складе (если это товар)
		// Предположим, что для билетов (TICKET) сток не так важен или обрабатывается иначе.
		if productPb.Type == pb.ProductTypeProto_MERCHANDISE { // Используем enum из pb
			if productPb.GetStock() < int32(itemInput.Quantity) {
				return nil, fmt.Errorf("%w: недостаточно продукта '%s' (ID: %s) на складе. В наличии: %d, запрошено: %d",
					ErrInsufficientStock, productPb.GetName(), itemInput.ProductID, productPb.GetStock(), itemInput.Quantity)
			}
		}

		orderItem := models.OrderItem{
			ProductID: itemInput.ProductID,
			Quantity:  itemInput.Quantity,
			Price:     productPb.GetPrice(), // Цена берется из ProductService на момент заказа
		}
		orderItems = append(orderItems, orderItem)
		totalPrice += productPb.GetPrice() * float64(itemInput.Quantity)
	}

	// 4. Создание объекта заказа
	order := &models.Order{
		UserID:     input.UserID,
		Items:      orderItems,
		TotalPrice: totalPrice,
		Status:     models.StatusPending, // Начальный статус
	}

	// 5. Сохранение заказа в репозитории (включая транзакцию для заказа и позиций)
	createdOrder, err := uc.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		// TODO: Здесь можно добавить логирование деталей ошибки
		return nil, fmt.Errorf("%w: %v", ErrCreateOrderFailed, err)
	}

	// 6. TODO: Уменьшение стока товаров в ProductService после успешного создания заказа.
	// Это должно быть сделано атомарно или с компенсационными транзакциями.
	// Например, для каждой позиции заказа:
	// uc.productClient.DecreaseStock(ctx, &pb.DecreaseStockRequest{ProductId: item.ProductID, Quantity: int32(item.Quantity)})
	// Если эта операция не удастся, нужно либо откатить создание заказа, либо пометить заказ как проблемный.

	return createdOrder, nil
}

// GetOrderByID находит заказ по ID.
func (uc *orderUsecase) GetOrderByID(ctx context.Context, orderID string) (*models.Order, error) {
	order, err := uc.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err // Ошибка репозитория
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	// TODO: Можно обогатить заказ информацией о пользователе и продуктах, сделав gRPC вызовы
	// к UserService и ProductService, если это необходимо для ответа.
	return order, nil
}

// ListOrdersByUserID возвращает список заказов для пользователя.
func (uc *orderUsecase) ListOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	// Сначала проверим, существует ли пользователь (опционально, но хорошая практика)
	_, err := uc.userClient.GetUser(ctx, &pb.GetUserRequest{Id: userID})
	if err != nil {
		return nil, fmt.Errorf("ошибка при проверке пользователя ID '%s' для списка заказов: %w; %w", userID, err, ErrUserNotFoundForOrder)
	}

	orders, err := uc.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if orders == nil {
		return []*models.Order{}, nil // Возвращаем пустой слайс, если заказов нет
	}
	// TODO: Обогащение каждого заказа информацией о продуктах, если необходимо.
	return orders, nil
}

// UpdateOrderStatus обновляет статус заказа.
func (uc *orderUsecase) UpdateOrderStatus(ctx context.Context, orderID string, status models.OrderStatus) (*models.Order, error) {
	// Валидация нового статуса (можно добавить более сложную логику переходов статусов)
	if status == "" {
		return nil, fmt.Errorf("%w: новый статус не может быть пустым", ErrInvalidOrderInput)
	}

	updatedOrder, err := uc.orderRepo.UpdateOrderStatus(ctx, orderID, status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || updatedOrder == nil { // sql.ErrNoRows может быть не возвращен явно, если UpdateOrderStatus возвращает nil,nil
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	// TODO: Отправка уведомлений пользователю об изменении статуса заказа (через NotificationService, если он есть)
	// TODO: Если статус заказа "PAID", возможно, нужно инициировать процесс уменьшения стока, если это не сделано ранее.
	return updatedOrder, nil
}
