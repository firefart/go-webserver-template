package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	CacheHits   *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		CacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits",
				Help: "Cache hits per cache",
			},
			[]string{"cache_name"},
		),
		CacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses",
				Help: "Cache misses per cache",
			},
			[]string{"cache_name"},
		),
	}
	if err := reg.Register(m.CacheHits); err != nil {
		return nil, err
	}
	if err := reg.Register(m.CacheMisses); err != nil {
		return nil, err
	}

	return m, nil
}
