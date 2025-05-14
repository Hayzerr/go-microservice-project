
package grpc

import (
	"context"
	"errors"

	// ВАЖНО: Замените 'your_order_module_path' и 'your_pb_module_path'
	// на актуальные пути к вашим модулям.
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/models"
	"github.com/Hayzerr/go-microservice-project/order-service/internal/order/usecase"
	pb "github.com/Hayzerr/go-microservice-project/pb" // Сгенерированные proto-файлы

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	// "google.golang.org/protobuf/types/known/emptypb" // Если будут методы, возвращающие Empty
)

// OrderGRPCHandler реализует gRPC сервер для OrderService.
type OrderGRPCHandler struct {
	pb.UnimplementedOrderServiceServer // Встраивание для обратной совместимости
	orderUsecase                   usecase.OrderUsecase
}

// NewOrderGRPCHandler создает новый экземпляр OrderGRPCHandler.
func NewOrderGRPCHandler(uc usecase.OrderUsecase) *OrderGRPCHandler {
	return &OrderGRPCHandler{orderUsecase: uc}
}

// --- Вспомогательные функции для преобразования ---

func mapOrderStatusToProto(status models.OrderStatus) pb.OrderStatusProto {
	switch status {
	case models.StatusPending:
		return pb.OrderStatusProto_PENDING
	case models.StatusPaid:
		return pb.OrderStatusProto_PAID
	case models.StatusProcessing:
		return pb.OrderStatusProto_PROCESSING
	case models.StatusShipped:
		return pb.OrderStatusProto_SHIPPED
	case models.StatusDelivered:
		return pb.OrderStatusProto_DELIVERED
	case models.StatusCompleted:
		return pb.OrderStatusProto_COMPLETED
	case models.StatusCancelled:
		return pb.OrderStatusProto_CANCELLED
	case models.StatusFailed:
		return pb.OrderStatusProto_FAILED
	default:
		return pb.OrderStatusProto_ORDER_STATUS_PROTO_UNSPECIFIED
	}
}

func mapProtoToOrderStatus(statusProto pb.OrderStatusProto) models.OrderStatus {
	switch statusProto {
	case pb.OrderStatusProto_PENDING:
		return models.StatusPending
	case pb.OrderStatusProto_PAID:
		return models.StatusPaid
	case pb.OrderStatusProto_PROCESSING:
		return models.StatusProcessing
	case pb.OrderStatusProto_SHIPPED:
		return models.StatusShipped
	case pb.OrderStatusProto_DELIVERED:
		return models.StatusDelivered
	case pb.OrderStatusProto_COMPLETED:
		return models.StatusCompleted
	case pb.OrderStatusProto_CANCELLED:
		return models.StatusCancelled
	case pb.OrderStatusProto_FAILED:
		return models.StatusFailed
	default:
		return "" // Или вернуть ошибку / статус по умолчанию
	}
}

func mapOrderItemModelToProto(item *models.OrderItem) *pb.OrderItemProto {
	if item == nil {
		return nil
	}
	return &pb.OrderItemProto{
		Id:        item.ID,
		ProductId: item.ProductID,
		Quantity:  int32(item.Quantity),
		Price:     item.Price,
	}
}

func mapOrderModelToProto(order *models.Order) *pb.OrderProto {
	if order == nil {
		return nil
	}
	itemsProto := make([]*pb.OrderItemProto, len(order.Items))
	for i, item := range order.Items {
		itemsProto[i] = mapOrderItemModelToProto(&item)
	}
	return &pb.OrderProto{
		Id:         order.ID,
		UserId:     order.UserID,
		Items:      itemsProto,
		TotalPrice: order.TotalPrice,
		Status:     mapOrderStatusToProto(order.Status),
		CreatedAt:  timestamppb.New(order.CreatedAt),
		UpdatedAt:  timestamppb.New(order.UpdatedAt),
	}
}

// --- Реализация RPC методов ---

// CreateOrder обрабатывает gRPC запрос на создание заказа.
func (h *OrderGRPCHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "UserID не может быть пустым")
	}
	if len(req.GetItems()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Заказ должен содержать хотя бы одну позицию")
	}

	itemsInput := make([]usecase.CreateOrderItemInput, len(req.GetItems()))
	for i, itemProto := range req.GetItems() {
		if itemProto.GetProductId() == "" || itemProto.GetQuantity() <= 0 {
			return nil, status.Errorf(codes.InvalidArgument, "ProductID и Quantity (>0) обязательны для каждой позиции заказа")
		}
		itemsInput[i] = usecase.CreateOrderItemInput{
			ProductID: itemProto.GetProductId(),
			Quantity:  int(itemProto.GetQuantity()),
		}
	}

	createInput := usecase.CreateOrderInput{
		UserID: req.GetUserId(),
		Items:  itemsInput,
	}

	order, err := h.orderUsecase.CreateOrder(ctx, createInput)
	if err != nil {
		// Обработка специфичных ошибок из usecase
		if errors.Is(err, usecase.ErrInvalidOrderInput) {
			return nil, status.Errorf(codes.InvalidArgument, "Некорректные данные заказа: %v", err)
		}
		if errors.Is(err, usecase.ErrUserNotFoundForOrder) {
			return nil, status.Errorf(codes.NotFound, "Пользователь для заказа не найден: %v", err)
		}
		if errors.Is(err, usecase.ErrProductNotFoundForOrder) {
			return nil, status.Errorf(codes.NotFound, "Один или несколько продуктов не найдены: %v", err)
		}
		if errors.Is(err, usecase.ErrInsufficientStock) {
			return nil, status.Errorf(codes.FailedPrecondition, "Недостаточно товара на складе: %v", err)
		}
		if errors.Is(err, usecase.ErrCreateOrderFailed) {
			return nil, status.Errorf(codes.Internal, "Не удалось создать заказ: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Внутренняя ошибка при создании заказа: %v", err)
	}

	return &pb.CreateOrderResponse{Order: mapOrderModelToProto(order)}, nil
}

// GetOrder обрабатывает gRPC запрос на получение заказа по ID.
func (h *OrderGRPCHandler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	orderID := req.GetOrderId()
	if orderID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "OrderID не может быть пустым")
	}

	order, err := h.orderUsecase.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, usecase.ErrOrderNotFound) {
			return nil, status.Errorf(codes.NotFound, "Заказ не найден: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Ошибка при получении заказа: %v", err)
	}

	return &pb.GetOrderResponse{Order: mapOrderModelToProto(order)}, nil
}

// ListOrdersByUserID обрабатывает gRPC запрос на получение списка заказов пользователя.
func (h *OrderGRPCHandler) ListOrdersByUserID(ctx context.Context, req *pb.ListOrdersByUserIDRequest) (*pb.ListOrdersResponse, error) {
	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "UserID не может быть пустым")
	}

	orders, err := h.orderUsecase.ListOrdersByUserID(ctx, userID)
	if err != nil {
		// Обработка специфичных ошибок, например, если пользователь не найден (хотя usecase это обрабатывает)
		if errors.Is(err, usecase.ErrUserNotFoundForOrder) {
			return nil, status.Errorf(codes.NotFound, "Пользователь для списка заказов не найден: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Ошибка при получении списка заказов: %v", err)
	}

	pbOrders := make([]*pb.OrderProto, len(orders))
	for i, o := range orders {
		pbOrders[i] = mapOrderModelToProto(o)
	}

	return &pb.ListOrdersResponse{Orders: pbOrders}, nil
}

// UpdateOrderStatus обрабатывает gRPC запрос на обновление статуса заказа.
func (h *OrderGRPCHandler) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.UpdateOrderStatusResponse, error) {
	orderID := req.GetOrderId()
	newStatusProto := req.GetNewStatus()

	if orderID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "OrderID не может быть пустым")
	}
	if newStatusProto == pb.OrderStatusProto_ORDER_STATUS_PROTO_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "Новый статус заказа не указан или некорректен")
	}

	newStatusModel := mapProtoToOrderStatus(newStatusProto)
	if newStatusModel == "" { // Проверка, что преобразование прошло успешно
		return nil, status.Errorf(codes.InvalidArgument, "Передан некорректный новый статус заказа")
	}


	order, err := h.orderUsecase.UpdateOrderStatus(ctx, orderID, newStatusModel)
	if err != nil {
		if errors.Is(err, usecase.ErrOrderNotFound) {
			return nil, status.Errorf(codes.NotFound, "Заказ для обновления статуса не найден: %v", err)
		}
		if errors.Is(err, usecase.ErrInvalidOrderInput){ // Если usecase вернет такую ошибку для статуса
			return nil, status.Errorf(codes.InvalidArgument, "Некорректный новый статус: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Ошибка при обновлении статуса заказа: %v", err)
	}

	return &pb.UpdateOrderStatusResponse{Order: mapOrderModelToProto(order)}, nil
}
