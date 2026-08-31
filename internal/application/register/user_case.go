// Package register реализует сценарий самостоятельной регистрации пользователя.
package register

import (
	"context"
	"fmt"

	"github.com/ZheglY/Interview-Material/internal/application/port"
	"github.com/ZheglY/Interview-Material/internal/domain"
)

// Command содержит входные данные сценария Register до преобразования в domain types.
type Command struct {
	Email string
	Password string
}

// Result содержит публичный результат успешно выполненной регистрации.
type Result struct {
	User domain.User
}

// UseCase координирует доменные правила и внешние зависимости регистрации.
type UseCase struct {
	users port.UserRepository
	passwordHasher port.PasswordHasher
	identifiers port.IDGenerator
	clock port.Clock
}

// New создаёт Register use case только с полностью переданными зависимостями.
func New(
	users port.UserRepository,
	passwordHasher port.PasswordHasher,
	identifiers port.IDGenerator,
	clock port.Clock,
) (*UseCase, error) {
	if users == nil || passwordHasher == nil || identifiers == nil || clock == nil {
		return nil, ErrInvalidDependencies
	}

	return &UseCase{
		users: users,
		passwordHasher: passwordHasher,
		identifiers: identifiers,
		clock: clock,
	}, nil
}

// Execute регистрирует пользователя или возвращает безопасную доменную ошибку.
func (useCase *UseCase) Execute(ctx context.Context, command Command) (Result, error) {
	// Не начинаем ресурсоёмкое хеширование, если клиент уже отменил запрос.
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	email, err := domain.NewEmail(command.Email)
	if err != nil {
		return Result{}, err
	}

	password, err := domain.NewPassword(command.Password)
	if err != nil {
		return Result{}, domain.ErrorForPasswordPolicy(err)
	}

	// Хеш вычисляется до repository, а исходный пароль не передаётся дальше.
	passwordHash, err := useCase.passwordHasher.Hash(password)
	if err != nil {
		return Result{}, fmt.Errorf("вычислить хеш пароля: %w", err)
	}

	identifier, err := useCase.identifiers.NewID()
	if err != nil {
		return Result{}, fmt.Errorf("создать идентификатор пользователя: %w", err)
	}

	user, err := domain.NewUser(
		identifier,
		email,
		useCase.clock.Now(),
	)
	if err != nil {
		return Result{}, err
	}

	if err := useCase.users.Create(ctx, user, passwordHash); err != nil {
		return Result{}, err
	}

	return Result{User: user}, nil
}
