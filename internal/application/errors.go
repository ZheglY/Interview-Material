package application

import "errors"

// ErrEmailAlreadyRegistered — прикладной конфликт уникальности email.
// Он принадлежит application-слою, потому что domain User сам по себе не
// знает о хранилище и не отвечает за проверку уникальности записей.
var ErrEmailAlreadyRegistered = errors.New("email уже зарегистрирован")
