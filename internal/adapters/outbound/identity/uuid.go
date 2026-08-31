// Package identity реализует генераторы идентификаторов для внешних адаптеров.
package identity

import "github.com/google/uuid"

// UUIDGenerator создаёт случайные UUID v4 для новых пользователей.
type UUIDGenerator struct {}

func NewUUIDGenerator() UUIDGenerator {
	return UUIDGenerator{}
}

// NewID возвращает UUID или ошибку источника криптографической случайности.
func (UUIDGenerator) NewID() (string, error) {
	identifier, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return identifier.String(), nil
}

//  1. Use case зависит от порта IDGenerator, а не от библиотеки github.com/google/uuid.
//  2. Bootstrap подставляет эту production-реализацию при запуске процесса.
