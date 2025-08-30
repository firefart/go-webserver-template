package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Metrics struct {
	CacheHits       *prometheus.CounterVec
	CacheMisses     *prometheus.CounterVec
	RequestCount    *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	RequestSize     *prometheus.HistogramVec
	ResponseSize    *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer, opts ...OptionsMetricsFunc) (*Metrics, error) {
	m := &Metrics{
		CacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Cache hits per cache",
			},
			[]string{"cache_name"},
		),
		CacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Cache misses per cache",
			},
			[]string{"cache_name"},
		),
	}
	// also add the default collectors
	if err := reg.Register(collectors.NewGoCollector()); err != nil {
		return nil, fmt.Errorf("failed to register go collector: %w", err)
	}
	if err := reg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return nil, fmt.Errorf("failed to register process collector: %w", err)
	}
	if err := reg.Register(m.CacheHits); err != nil {
		return nil, fmt.Errorf("failed to register cache hits metric: %w", err)
	}
	if err := reg.Register(m.CacheMisses); err != nil {
		return nil, fmt.Errorf("failed to register cache misses metric: %w", err)
	}

	for _, o := range opts {
		if err := o(m, reg); err != nil {
			return nil, err
		}
	}

	return m, nil
}
