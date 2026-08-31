// Package memory содержит временные in-memory реализации application ports.
package memory

import (
	"context"
	"sync"

	"github.com/ZheglY/Interview-Material/internal/application"
	"github.com/ZheglY/Interview-Material/internal/domain"
)

// UserRepository хранит пользователей в памяти процесса только для ранних этапов.
type UserRepository struct {
	mutex       sync.RWMutex
	usersByMail map[string]userRecord
}

// userRecord связывает публичного User с private password hash внутри repository.
type userRecord struct {
	user         domain.User
	passwordHash domain.PasswordHash
}

// NewUserRepository создаёт пустой потокобезопасный repository.
func NewUserRepository() *UserRepository {
	return &UserRepository{
		usersByMail: make(map[string]userRecord),
	}
}

// Create атомарно сохраняет пользователя или сообщает о повторном email.
func (repository *UserRepository) Create(
	ctx context.Context,
	user domain.User,
	passwordHash domain.PasswordHash,
) error {
	// Repository не начинает mutation, если gRPC-клиент уже отменил работу.
	if err := ctx.Err(); err != nil {
		return err
	}

	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}

	key := user.Email().Value()
	if _, exists := repository.usersByMail[key]; exists {
		return application.ErrEmailAlreadyRegistered
	}

	repository.usersByMail[key] = userRecord{
		user:         user,
		passwordHash: passwordHash,
	}
	return nil
}

// Count возвращает число пользователей и нужен только диагностическим unit-тестам.
func (repository *UserRepository) Count() int {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()
	return len(repository.usersByMail)
}

// Общая логика файла:
//  1. map заменяет PostgreSQL только на учебном этапе.
//  2. mutex сохраняет атомарность проверки и вставки при параллельных Register.
//  3. По ключу email нельзя создать двух пользователей.
//  4. На Этапе 5 этот adapter будет заменён PostgreSQL repository без изменения use case.
