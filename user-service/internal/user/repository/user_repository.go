package repository

import (
	"context"
	"database/sql"
	"errors" // Для стандартных ошибок, таких как sql.ErrNoRows
	"time"

	// ВАЖНО: Замените 'your_project_module' на имя вашего модуля из go.mod
	"github.com/Hayzerr/go-microservice-project/user-service/internal/user/models"

	"github.com/google/uuid" // Для генерации UUID в качестве ID
)

// UserRepository определяет интерфейс для взаимодействия с хранилищем данных пользователей.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) (*models.User, error)
	Delete(ctx context.Context, id string) error // Новый метод
	// TODO: Добавьте другие методы по мере необходимости (List и т.д.)
}

// postgresUserRepository реализует UserRepository для PostgreSQL.
type postgresUserRepository struct {
	db *sql.DB // Пул соединений с базой данных
}

// NewPostgresUserRepository создает новый экземпляр postgresUserRepository.
func NewPostgresUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

// Create создает новую запись пользователя в базе данных.
func (r *postgresUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	user.ID = uuid.NewString()
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	query := `INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
			   VALUES ($1, $2, $3, $4, $5, $6)
			   RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, user.ID, user.Username, user.Email, user.Password, user.CreatedAt, user.UpdatedAt).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}
	createdUser := *user
	createdUser.Password = ""
	return &createdUser, nil
}

// GetByID извлекает пользователя из базы данных по его ID.
func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, username, email, password_hash, created_at, updated_at
			   FROM users
			   WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// GetByEmail извлекает пользователя из базы данных по его email.
func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, username, email, password_hash, created_at, updated_at
			   FROM users
			   WHERE email = $1`

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// Update обновляет существующую запись пользователя в базе данных.
func (r *postgresUserRepository) Update(ctx context.Context, user *models.User) (*models.User, error) {
	user.UpdatedAt = time.Now().UTC()
	query := `UPDATE users
			   SET username = $1, email = $2, updated_at = $3
			   WHERE id = $4
			   RETURNING id, username, email, password_hash, created_at, updated_at`

	updatedUser := &models.User{}
	err := r.db.QueryRowContext(ctx, query, user.Username, user.Email, user.UpdatedAt, user.ID).Scan(
		&updatedUser.ID,
		&updatedUser.Username,
		&updatedUser.Email,
		&updatedUser.Password,
		&updatedUser.CreatedAt,
		&updatedUser.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return updatedUser, nil
}

// Delete удаляет пользователя из базы данных по его ID.
func (r *postgresUserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows // Используем sql.ErrNoRows, чтобы указать, что пользователь не был найден для удаления
	}

	return nil
}

// Пример схемы таблицы 'users' для PostgreSQL:
/*
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
*/
