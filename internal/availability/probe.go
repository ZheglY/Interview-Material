/* 
Package availability хранит минимальное состояние готовности процесса.
Он нужен, чтобы сообщать Kubernetes или другому оркестратору, 
можно ли прямо сейчас направлять на процесс новые HTTP-запросы.
*/

package availability

import "sync/atomic"

// Флажок который безопасно сообщает готовность между несколькими goroutine
// переводится как "проверка" или "датчик"
type Probe struct {
	ready atomic.Bool
}

func NewProbe() *Probe {
	return &Probe{}
}

// MarkReady сообщает оркестратору, что процесс готов принимать трафик.
func (probe *Probe) MarkReady() {
	probe.ready.Store(true)
}

// MarkNotReady запрещает отправлять новый трафик во время остановки.
func (probe *Probe) MarkNotReady() {
	probe.ready.Store(false)
}

// IsReady возвращает текущее состояние без блокировок mutex.
func (probe *Probe) IsReady() bool {
	return probe.ready.Load()
}


// Общая логика файла:
//  1. Новый процесс ещё не готов и возвращает false.
//  2. После успешного запуска listeners bootstrap переключает состояние в true.
//  3. Перед graceful shutdown состояние снова становится false.


/*
Запуск приложения
      ↓
ready = false
      ↓
Подключение к БД, запуск HTTP-сервера
      ↓
ready = true
      ↓
Приложение обрабатывает запросы
      ↓
Получен сигнал остановки
      ↓
ready = false
      ↓
Завершение оставшихся запросов
      ↓
Остановка процесса
*/