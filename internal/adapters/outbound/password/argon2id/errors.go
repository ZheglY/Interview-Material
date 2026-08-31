package argon2id

import "errors"

// ErrInvalidParameters означает небезопасную или неполную конфигурацию Argon2id.
var ErrInvalidParameters = errors.New("некорректные параметры Argon2id")
