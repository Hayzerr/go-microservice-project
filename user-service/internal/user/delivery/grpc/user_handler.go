package grpc

import (
	"context"
	"errors" // Для проверки типов ошибок из usecase

	// ВАЖНО: Замените 'your_project_module' на имя вашего модуля из go.mod
	pb "github.com/Hayzerr/go-microservice-project/pb" // Сгенерированные proto-файлы
	"github.com/Hayzerr/go-microservice-project/user-service/internal/user/models"
	"github.com/Hayzerr/go-microservice-project/user-service/internal/user/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb" // Для преобразования time.Time в google.protobuf.Timestamp
)

// UserGRPCHandler реализует gRPC сервер для UserService.
type UserGRPCHandler struct {
	pb.UnimplementedUserServiceServer // Встраивание для обратной совместимости
	userUsecase                       usecase.UserUsecase
}

// NewUserGRPCHandler создает новый экземпляр UserGRPCHandler.
func NewUserGRPCHandler(uc usecase.UserUsecase) *UserGRPCHandler {
	return &UserGRPCHandler{userUsecase: uc}
}

// mapUserModelToProto преобразует модель User в proto-сообщение User.
func mapUserModelToProto(user *models.User) *pb.User {
	if user == nil {
		return nil
	}
	return &pb.User{
		Id:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt), // Преобразование time.Time
		UpdatedAt: timestamppb.New(user.UpdatedAt), // Преобразование time.Time
	}
}

// CreateUser обрабатывает gRPC запрос на создание пользователя.
func (h *UserGRPCHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	// Валидация входных данных (можно добавить больше проверок)
	if req.GetUsername() == "" || req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Имя пользователя, email и пароль не могут быть пустыми")
	}

	user, err := h.userUsecase.RegisterUser(ctx, req.GetUsername(), req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, usecase.ErrEmailExists) {
			return nil, status.Errorf(codes.AlreadyExists, "Пользователь с таким email уже существует: %v", err)
		}
		if errors.Is(err, usecase.ErrPasswordTooShort) {
			return nil, status.Errorf(codes.InvalidArgument, "Пароль слишком короткий: %v", err)
		}
		// TODO: Добавить обработку других специфичных ошибок usecase
		return nil, status.Errorf(codes.Internal, "Ошибка при создании пользователя: %v", err)
	}

	return &pb.CreateUserResponse{User: mapUserModelToProto(user)}, nil
}

// GetUser обрабатывает gRPC запрос на получение пользователя по ID.
func (h *UserGRPCHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID пользователя не может быть пустым")
	}

	user, err := h.userUsecase.FindUserByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "Пользователь не найден: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Ошибка при получении пользователя: %v", err)
	}

	return &pb.GetUserResponse{User: mapUserModelToProto(user)}, nil
}

// UpdateUser обрабатывает gRPC запрос на обновление пользователя.
func (h *UserGRPCHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	userID := req.GetId()
	if userID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID пользователя для обновления не может быть пустым")
	}

	// Подготовка входных данных для usecase.UpdateUser
	updateInput := usecase.UpdateUserInput{}

	if req.Username != nil {
		// Если используется wrappers.StringValue, то req.GetUsername() вернет сам *wrapperspb.StringValue
		// а его значение .GetValue()
		// Для простоты, если вы решили не использовать wrapperspb в .proto,
		// то просто проверяйте req.GetUsername() != ""
		usernameVal := req.GetUsername().GetValue() // Для wrapperspb.StringValue
		updateInput.Username = &usernameVal
	}
	if req.Email != nil {
		emailVal := req.GetEmail().GetValue() // Для wrapperspb.StringValue
		updateInput.Email = &emailVal
	}

	// Проверка, есть ли вообще что обновлять
	if updateInput.Username == nil && updateInput.Email == nil {
		// Можно вернуть ошибку или текущее состояние пользователя
		// Здесь мы возвращаем ошибку, т.к. запрос на обновление без данных для обновления бессмысленен
		return nil, status.Errorf(codes.InvalidArgument, "Нет данных для обновления")
	}

	updatedUser, err := h.userUsecase.UpdateUser(ctx, userID, updateInput)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			return nil, status.Errorf(codes.NotFound, "Пользователь для обновления не найден: %v", err)
		case errors.Is(err, usecase.ErrEmailExists):
			return nil, status.Errorf(codes.AlreadyExists, "Новый email уже используется другим пользователем: %v", err)
		// TODO: Обработать другие специфичные ошибки usecase, если они появятся
		default:
			return nil, status.Errorf(codes.Internal, "Ошибка при обновлении пользователя: %v", err)
		}
	}

	return &pb.UpdateUserResponse{User: mapUserModelToProto(updatedUser)}, nil
}

// TODO: Реализуйте другие gRPC методы, определенные в вашем user.proto
// Например, AuthenticateUser, DeleteUser и т.д.
