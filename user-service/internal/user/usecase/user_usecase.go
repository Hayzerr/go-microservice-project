package usecase

import (
	"context"
	"errors" // Для создания кастомных ошибок
	"time"   // Для обновления UpdatedAt

	// ВАЖНО: Замените 'your_project_module' на имя вашего модуля из go.mod
	"github.com/Hayzerr/go-microservice-project/user-service/internal/user/models"
	"github.com/Hayzerr/go-microservice-project/user-service/internal/user/repository"

	"golang.org/x/crypto/bcrypt" // Для хеширования паролей
)

var (
	ErrUserNotFound       = errors.New("пользователь не найден")
	ErrEmailExists        = errors.New("пользователь с таким email уже существует")
	ErrInvalidCredentials = errors.New("неверные учетные данные")
	ErrPasswordTooShort   = errors.New("пароль слишком короткий")
	ErrUpdateConflict     = errors.New("конфликт при обновлении данных пользователя")
	// TODO: Добавьте другие специфичные для бизнес-логики ошибки
)

// PasswordHasher определяет интерфейс для хеширования и проверки паролей.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}

// BcryptPasswordHasher реализует PasswordHasher с использованием bcrypt.
type BcryptPasswordHasher struct {
	cost int // Стоимость хеширования для bcrypt (например, bcrypt.DefaultCost)
}

// NewBcryptPasswordHasher создает новый экземпляр BcryptPasswordHasher.
// Рекомендуемая стоимость cost: bcrypt.DefaultCost (обычно 10) или выше.
func NewBcryptPasswordHasher(cost int) PasswordHasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptPasswordHasher{cost: cost}
}

// Hash генерирует хеш bcrypt для заданного пароля.
func (h *BcryptPasswordHasher) Hash(password string) (string, error) {
	// Проверка минимальной длины пароля перед хешированием
	if len(password) < 8 { // Пример: минимальная длина 8 символов
		return "", ErrPasswordTooShort
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	return string(bytes), err
}

// Compare сравнивает хеш bcrypt с паролем.
func (h *BcryptPasswordHasher) Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// UpdateUserInput определяет структуру для входных данных при обновлении пользователя.
// Позволяет обновлять только определенные поля. Пароль здесь не обновляется.
type UpdateUserInput struct {
	Username *string // Указатель, чтобы различать пустое значение и отсутствие поля
	Email    *string
}

// UserUsecase определяет интерфейс для бизнес-логики, связанной с пользователями.
type UserUsecase interface {
	RegisterUser(ctx context.Context, username, email, rawPassword string) (*models.User, error)
	AuthenticateUser(ctx context.Context, email, rawPassword string) (*models.User, error)
	FindUserByID(ctx context.Context, id string) (*models.User, error)
	UpdateUser(ctx context.Context, id string, input UpdateUserInput) (*models.User, error)
	// TODO: Добавьте другие методы бизнес-логики (например, DeleteUser, ChangePassword)

	ListUsers(ctx context.Context) ([]*models.User, error)
	DeleteUser(ctx context.Context, id string) error
}

type userUsecase struct {
	userRepo       repository.UserRepository
	passwordHasher PasswordHasher // Интерфейс для хеширования паролей
}

// NewUserUsecase создает новый экземпляр userUsecase.
func NewUserUsecase(userRepo repository.UserRepository, hasher PasswordHasher) UserUsecase {
	return &userUsecase{
		userRepo:       userRepo,
		passwordHasher: hasher,
	}
}

// RegisterUser регистрирует нового пользователя.
func (uc *userUsecase) RegisterUser(ctx context.Context, username, email, rawPassword string) (*models.User, error) {
	existingUser, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrEmailExists
	}

	hashedPassword, err := uc.passwordHasher.Hash(rawPassword)
	if err != nil {
		if errors.Is(err, ErrPasswordTooShort) {
			return nil, ErrPasswordTooShort
		}
		return nil, err
	}

	user := &models.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
	}

	createdUser, err := uc.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	return createdUser, nil
}

// AuthenticateUser аутентифицирует пользователя по email и паролю.
func (uc *userUsecase) AuthenticateUser(ctx context.Context, email, rawPassword string) (*models.User, error) {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	err = uc.passwordHasher.Compare(user.Password, rawPassword)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	authenticatedUser := *user
	authenticatedUser.Password = ""
	return &authenticatedUser, nil
}

// FindUserByID находит пользователя по его ID.
func (uc *userUsecase) FindUserByID(ctx context.Context, id string) (*models.User, error) {
	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	foundUser := *user
	foundUser.Password = ""
	return &foundUser, nil
}

// UpdateUser обновляет данные существующего пользователя.
// Пароль не обновляется этим методом. Для смены пароля должен быть отдельный метод.
func (uc *userUsecase) UpdateUser(ctx context.Context, id string, input UpdateUserInput) (*models.User, error) {
	// 1. Получаем текущего пользователя
	currentUser, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err // Ошибка репозитория
	}
	if currentUser == nil {
		return nil, ErrUserNotFound
	}

	// 2. Обновляем поля, если они предоставлены во входных данных
	userToUpdate := *currentUser // Копируем, чтобы не изменять currentUser напрямую до успешного обновления
	changed := false

	if input.Username != nil && *input.Username != currentUser.Username {
		userToUpdate.Username = *input.Username
		changed = true
	}

	if input.Email != nil && *input.Email != currentUser.Email {
		// Если email меняется, нужно проверить, не занят ли новый email
		existingUserWithNewEmail, err := uc.userRepo.GetByEmail(ctx, *input.Email)
		if err != nil {
			return nil, err // Ошибка репозитория
		}
		if existingUserWithNewEmail != nil && existingUserWithNewEmail.ID != id {
			return nil, ErrEmailExists // Новый email уже используется другим пользователем
		}
		userToUpdate.Email = *input.Email
		changed = true
	}

	// Если ничего не изменилось, просто возвращаем текущего пользователя
	if !changed {
		updatedUser := *currentUser
		updatedUser.Password = "" // Убираем пароль перед возвратом
		return &updatedUser, nil
	}

	// 3. Устанавливаем время обновления
	userToUpdate.UpdatedAt = time.Now().UTC()

	// 4. Вызываем метод репозитория для обновления
	// Предполагается, что в UserRepository есть метод Update
	// и он принимает *models.User для обновления.
	updatedUser, err := uc.userRepo.Update(ctx, &userToUpdate)
	if err != nil {
		// Здесь можно добавить обработку специфичных ошибок репозитория,
		// например, если возник конфликт версий или другая проблема при обновлении.
		// В данном примере просто пробрасываем ошибку.
		// Можно также проверить на sql.ErrNoRows, если Update возвращает его,
		// хотя это маловероятно, если мы только что получили пользователя по ID.
		return nil, err
	}
	updatedUser.Password = "" // Убираем пароль перед возвратом
	return updatedUser, nil
}

func (uc *userUsecase) ListUsers(ctx context.Context) ([]*models.User, error) {
	users, err := uc.userRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		u.Password = ""
	}
	return users, nil
}

func (uc *userUsecase) DeleteUser(ctx context.Context, id string) error {
	return uc.userRepo.Delete(ctx, id)
}
