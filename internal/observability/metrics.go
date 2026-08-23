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
	// Реестр — объект, в котором хранятся зарегистрированные collectors:
	registry := prometheus.NewRegistry()

	// Process collector сообщает CPU, память и число файловых дескрипторов процесса.
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// GaugeVec хранит группу Gauge-метрик с одним именем, но разными label values.
	buildMetrics := prometheus.NewGaugeVec( //- имя метрики — auth_build_info;
		prometheus.GaugeOpts{
			Namespace: "auth",
			Name: "build_info",
			Help: "Сведение о версии запущенного auth-сервиса.",
		},
		[]string{"version", "commit", "build_time"}, 
	)

	//- version, commit, build_time — labels
	//- числовое значение метрики — 1.
	buildMetrics.WithLabelValues(buildinfo.Version, buildinfo.Commit, buildinfo.BuildTime).Set(1) 
	registry.MustRegister(buildMetrics)

	return registry
}

// Общая логика файла:
//  1. Для процесса создаётся отдельный Prometheus registry.
//  2. В него добавляются метрики Go runtime и операционной системы.
//  3. build_info позволяет увидеть версию бинарного файла через /metrics.

/*
NewRegistry()
      ↓
Создаёт пустой реестр
      ↓
Добавляет метрики процесса
      ↓
Добавляет auth_build_info
      ↓
Возвращает реестр
      ↓
HTTP endpoint /metrics использует этот реестр
      ↓
Prometheus периодически читает /metrics

*/
