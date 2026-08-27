package domain

import "time"

type User struct {
	id        string
	email     Email
	createdAt time.Time
}

func NewUser(id string, email Email, createdAt time.Time) (*User, error) {
	if id == "" {
		return &User{}, ErrInvalidUserID
	}

	if email.Value() == "" {
		return &User{}, ErrInvalidUserCreatedAt
	}

	if createdAt.IsZero() { // Функция из стандартной библиотки GO
		return &User{}, ErrInvalidUserCreatedAt
	}

	return &User{
		id: id,
		email: email,
		createdAt: createdAt.UTC(),
	}, nil
}

// обычная инкапсуляция
func (user *User) ID() string {
	return user.id
}

func (user *User) Email() Email {
	return user.email
}

func (user *User) CreatedAt() time.Time {
	return user.createdAt
}

//  1. User создаётся только через NewUser с проверенными обязательными полями.
//  2. Внутренние поля нельзя изменить напрямую после создания.
//  3. Время приводится к UTC на границе домена.
//  4. Секреты пользователя хранятся отдельно в persistence-модели.
