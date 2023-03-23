package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

type OperatorHealth struct {
	// operatorName is the name of the operator.
	operatorName string

	healthy   prometheus.Gauge
	degraded  prometheus.Gauge
	unhealthy prometheus.Gauge
	unknown   prometheus.Gauge
}

func NewOperatorHealth(operatorName string) *OperatorHealth {
	healthy := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "operator_lib",
		Name:      "operator_healthy",
		Help:      "Whether the operator is healthy. Value 1 means healthy, 0 means not healthy.",
		ConstLabels: map[string]string{
			"operator_name": operatorName,
		},
	})

	degraded := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "operator_lib",
		Name:      "operator_degraded",
		Help:      "Whether the operator is degraded. Value 1 means degraded, 0 means not degraded.",
		ConstLabels: map[string]string{
			"operator_name": operatorName,
		},
	})

	unhealthy := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "operator_lib",
		Name:      "operator_unhealthy",
		Help:      "Whether the operator is unhealthy. Value 1 means unhealthy, 0 means not unhealthy.",
		ConstLabels: map[string]string{
			"operator_name": operatorName,
		},
	})

	unknown := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "operator_lib",
		Name:      "operator_health_unknown",
		Help:      "Whether the operator health is unknown. Value 1 means unknown, 0 means not unknown.",
		ConstLabels: map[string]string{
			"operator_name": operatorName,
		},
	})

	m := &OperatorHealth{
		operatorName: operatorName,
		healthy:      healthy,
		degraded:     degraded,
		unhealthy:    unhealthy,
		unknown:      unknown,
	}
	m.MustSet(OperatorHealthUnknown)
	return m
}

func (m *OperatorHealth) Register(registry prometheus.Registerer) error {
	for _, v := range []prometheus.Gauge{m.healthy, m.degraded, m.unhealthy, m.unknown} {
		if err := registry.Register(v); err != nil {
			return err
		}
	}
	return nil
}

func (m *OperatorHealth) Collect(ch chan<- prometheus.Metric) {
	for _, v := range []prometheus.Collector{m.healthy, m.degraded, m.unhealthy, m.unknown} {
		v.Collect(ch)
	}
}

func (m *OperatorHealth) Describe(ch chan<- *prometheus.Desc) {
	for _, v := range []prometheus.Collector{m.healthy, m.degraded, m.unhealthy, m.unknown} {
		v.Describe(ch)
	}
}

type OperatorHealthState string

const (
	OperatorHealthHealthy   OperatorHealthState = "healthy"
	OperatorHealthDegraded  OperatorHealthState = "degraded"
	OperatorHealthUnhealthy OperatorHealthState = "unhealthy"
	OperatorHealthUnknown   OperatorHealthState = "unknown"
)

func (m *OperatorHealth) Set(status OperatorHealthState) error {
	switch status {
	case OperatorHealthHealthy:
		m.healthy.Set(1)
		m.degraded.Set(0)
		m.unhealthy.Set(0)
		m.unknown.Set(0)
	case OperatorHealthDegraded:
		m.healthy.Set(0)
		m.degraded.Set(1)
		m.unhealthy.Set(0)
		m.unknown.Set(0)
	case OperatorHealthUnhealthy:
		m.healthy.Set(0)
		m.degraded.Set(0)
		m.unhealthy.Set(1)
		m.unknown.Set(0)
	case OperatorHealthUnknown:
		m.healthy.Set(0)
		m.degraded.Set(0)
		m.unhealthy.Set(0)
		m.unknown.Set(1)
	default:
		return fmt.Errorf("unknown operator health status %q", status)
	}
	return nil
}

func (m *OperatorHealth) MustSet(status OperatorHealthState) {
	if err := m.Set(status); err != nil {
		panic(err)
	}
}
