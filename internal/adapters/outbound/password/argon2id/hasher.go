package argon2id

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/ZheglY/Interview-Material/internal/domain"
	"golang.org/x/crypto/argon2"
)

/*
passwordBytes — пароль;
salt — уникальная случайная соль;
Iterations — сколько раз выполнять вычисление;
Memory — сколько памяти использовать;
Parallelism — сколько параллельных потоков расчёта использовать;
KeyLength — длина итогового хеша в байтах.
*/
type Parameters struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

const (
	minimumMemoryKiB   = 19 * 1024
	minimumIterations  = 2
	minimumParallelism = 1
	minimumSaltLength  = 16
	minimumKeyLength   = 16
)

// DefaultParameters возвращает минимальную рекомендуемую OWASP конфигурацию Argon2id.
func DefaultParameters() Parameters {
	return Parameters{
		Memory:      19 * 1024, // 19 MiB в KiB — Argon2 API принимает именно KiB.
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// Hasher реализует application port PasswordHasher.
type Hasher struct {
	parameters Parameters
}

// New создаёт Hasher только с валидными параметрами вычислительной стоимости.
func New(parameters Parameters) (*Hasher, error) {
	if parameters.Memory < minimumMemoryKiB || parameters.Iterations < minimumIterations ||
		parameters.Parallelism < minimumParallelism || parameters.SaltLength < minimumSaltLength ||
		parameters.KeyLength < minimumKeyLength {
		return nil, ErrInvalidParameters
	}

	return &Hasher{parameters: parameters}, nil
}

func (hasher *Hasher) Hash(password domain.Password) (domain.PasswordHash, error) {
	// Уникальная соль не даёт одинаковым паролям образовать одинаковый hash.
	salt := make([]byte, hasher.parameters.SaltLength) // Создаётся массив байтов для соли. [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
	if _, err := rand.Read(salt); err != nil { // заполняет существующий массив случайными байтами.
		return domain.PasswordHash{}, fmt.Errorf("создать соль Argon2id: %w", err)
	}

	// Argon2 принимает []byte; копию очищаем сразу после получения ключа.
	passwordBytes := []byte(password.Value())
	defer clear(passwordBytes) // заполнит массив нулями
	
	derivedKey := argon2.IDKey(
		passwordBytes,
		salt,
		hasher.parameters.Iterations,
		hasher.parameters.Memory,
		hasher.parameters.Parallelism,
		hasher.parameters.KeyLength,
	)
	defer clear(derivedKey)

	// PHC encoding хранит алгоритм и параметры вместе с солью и ключом для будущей проверки.
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		hasher.parameters.Memory,
		hasher.parameters.Iterations,
		hasher.parameters.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(derivedKey),
	)

	return domain.NewPasswordHash(encoded)	
}

//  1. Hasher создаёт криптографически случайную соль для каждого пароля.
//  2. Argon2id получает пароль, соль и явные параметры стоимости.
//  3. Результат сериализуется в PHC-строку и хранится в repository.
//  4. Исходный пароль не возвращается и не записывается в логи.

/*
Пароль
  + случайная соль
  + параметры сложности
        ↓
     Argon2id
        ↓
   derivedKey (байты)
        ↓
PHC-строка с алгоритмом, параметрами, солью и хешем
        ↓
Сохранение в repository / БД
*/