// Package observability собирает техническую телеметрию приложения.
package observability

import (
	"github.com/ZheglY/Interview-Material/internal/buildinfo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

/*
Этот код создаёт отдельный реестр Prometheus-метрик для auth-сервиса и регистрирует в нём:
- технические метрики процесса: CPU, память, файловые дескрипторы и время запуска;
- информационную метрику о версии сборки приложения.

Сам код не запускает HTTP-сервер и не отправляет метрики в 
Prometheus. Он только подготавливает объект 
*prometheus.Registry. Позже этот объект нужно подключить к 
endpoint вроде /metrics.
*/

// NewRegistry создаёт изолированный реестр метрик для одного приложения.
func NewRegistry() *prometheus.Registry {
	// Собственный registry не зависит от глобального состояния сторонних пакетов.
	registry := prometheus.NewRegistry()

	// Process collector сообщает CPU, память и число файловых дескрипторов процесса.
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	buildMetics := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "auth",
			Name: "build_info",
			Help: "Сведение о версии запущенного auth-сурвиса.",
		},
		[]string{"version", "commit", "build_time"},
	)

	buildMetics.WithLabelValues(buildinfo.Version, buildinfo.Commit, buildinfo.BuildTime).Set(1)
	registry.MustRegister(buildMetics)

	return registry
}

// Общая логика файла:
//  1. Для процесса создаётся отдельный Prometheus registry.
//  2. В него добавляются метрики Go runtime и операционной системы.
//  3. build_info позволяет увидеть версию бинарного файла через /metrics.

/*
NewRegistry()
    ↓
создать пустой registry
    ↓
добавить process_* метрики
    ↓
создать auth_build_info
    ↓
записать version, commit и build_time
    ↓
вернуть registry
    ↓
подключить registry к /metrics
    ↓
Prometheus периодически читает /metrics
*/