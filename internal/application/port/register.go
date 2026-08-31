// Package port содержит интерфейсы, нужные application use case-ам.
package port

import (
	"context"
	"time"

	"github.com/ZheglY/Interview-Material/internal/domain"
)

// UserRepository сохраняет нового пользователя и его password hash атомарно.
// При нарушении уникальности email реализация возвращает
// application.ErrEmailAlreadyRegistered; технические ошибки БД наружу не протекают.
type UserRepository interface {
	Create(ctx context.Context, user domain.User, passwordHash domain.PasswordHash) error
}

// PasswordHasher создаёт адаптивный хеш без раскрытия алгоритма use case-у.
type PasswordHasher interface {
	Hash(password domain.Password) (domain.PasswordHash, error)
}

// IDGenerator выдаёт уникальные идентификаторы, не привязывая use case к UUID-библиотеке.
type IDGenerator interface {
	NewID() (string, error)
}

// Clock делает время создания пользователя детерминированным в unit-тестах.
type Clock interface {
	Now() time.Time
}

//  1. Application-слой владеет контрактами нужных ему зависимостей.
//  2. PostgreSQL, Argon2id и UUID находятся по другую сторону этих интерфейсов.
//  3. Тесты use case подставляют маленькие in-memory реализации портов.
