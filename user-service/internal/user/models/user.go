package models

import (
	"time"
)

// User представляет модель пользователя в системе.
type User struct {
	ID        string    `json:"id"`         // Уникальный идентификатор пользователя (например, UUID)
	Username  string    `json:"username"`   // Имя пользователя
	Email     string    `json:"email"`      // Адрес электронной почты (должен быть уникальным)
	Password  string    `json:"-"`          // Хеш пароля (не включаем в JSON ответы напрямую)
	CreatedAt time.Time `json:"created_at"` // Время создания записи пользователя
	UpdatedAt time.Time `json:"updated_at"` // Время последнего обновления записи пользователя
}

// Вы можете добавить сюда методы для структуры User, если это необходимо.
// Например, для валидации данных.
