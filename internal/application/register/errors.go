package register

import "errors"

// ErrInvalidDependencies означает ошибку сборки use case в composition root.
var ErrInvalidDependencies = errors.New("все зависимости Register use case обязательны")
