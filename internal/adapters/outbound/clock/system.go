// Package clock реализует источники времени для внешних адаптеров.
package clock

import "time"

// SystemClock возвращает фактическое системное время процесса.
type SystemClock struct {}

func NewSystemLock() SystemClock {
	return SystemClock{}
}

// Now возвращает текущее время сразу в UTC, используемом для persistence и API.
func (SystemClock) Now() time.Time{
	return time.Now().UTC()
}

//  1. Production-реализация обращается к системным часам ровно в одном месте.
//  2. Unit-тесты use case заменяют её детерминированным clockStub.
