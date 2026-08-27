package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// Политика пароля допускает длинные passphrase и ограничивает расход ресурсов.
	minPasswordLength = 12
	maxPasswordLength = 128
)

// Password хранит исходный пароль только в пределах одного application use case.
type Password struct {
	value string
}

// NewPassword проверяет минимальную политику без требований к специальным символам.
func NewPassword(raw string) (Password, error) {
	if !utf8.ValidString(raw) {
		return Password{}, ErrPasswordInvalidEncoding
	}
	length := utf8.RuneCountInString(raw)
	if length < minPasswordLength || strings.TrimSpace(raw) == "" {
		return Password{}, ErrPasswordTooShort
	}
	if length > maxPasswordLength {
		return Password{}, ErrPasswordTooLong
	}

	return Password{value: raw}, nil
}

// Value возвращает пароль только password hasher-у; Password не реализует fmt.Stringer.
func (password Password) Value() string {
	return password.value
}


// PasswordHash представляет уже вычисленный адаптивный хеш, готовый к persistence.
type PasswordHash struct {
	value string
}

// NewPasswordHash проверяет, что infrastructure adapter не вернул пустой результат.
func NewPasswordHash(value string) (PasswordHash, error) {
	if value == "" {
		return PasswordHash{}, ErrEmptyPasswordHash
	}
	return PasswordHash{value: value}, nil
}

// Value возвращает сериализованное представление хеша для repository.
func (hash PasswordHash) Value() string {
	return hash.value
}

// ErrorForPasswordPolicy превращает доменную ошибку в безопасное описание для transport.
func ErrorForPasswordPolicy(err error) error {
	if errors.Is(err, ErrPasswordTooShort) || errors.Is(err, ErrPasswordTooLong) {
		return fmt.Errorf("%w: длина должна быть от %d до %d символов", err, minPasswordLength, maxPasswordLength)
	}
	return err
}

// Общая логика файла:
//  1. Password создаётся только после проверки длины и непустого содержимого.
//  2. PasswordHash отделён от исходного пароля и используется для persistence.
//  3. Ошибки политики сохраняют исходную категорию через wrapping.
