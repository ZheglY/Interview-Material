package domain

import "errors"

var (
	ErrInvalidEmail = errors.New("некорректный email")
	ErrPasswordTooShort = errors.New("пароль слишком короткий")
	ErrPasswordTooLong = errors.New("пароль слишком длинный")
	ErrPasswordInvalidEncoding = errors.New("пароль содержит некорректную кодировку")
	ErrInvalidUserID = errors.New("идентификатор пользователя не должен быть пустым")
	ErrInvalidUserCreatedAt = errors.New("время создания пользователя не должно быть нулевым")
	ErrEmptyPasswordHash = errors.New("пустой хеш пароля")
)