package metric

import (
	"context"
	"expvar"
	"runtime/pprof"
	"time"
)

const (
	metricsMapName = "pprof"
)

var (
	profiles = func() []*pprof.Profile {
		pns := []string{"goroutine", "heap", "allocs", "threadcreate", "block", "mutex"}
		ps := make([]*pprof.Profile, 0, len(pns))
		for _, pn := range pns {
			if p := pprof.Lookup(pn); p != nil {
				ps = append(ps, p)
			}
		}
		return ps
	}()
	metrics = func(metricsMap *expvar.Map) map[string]*expvar.Int {
		m := make(map[string]*expvar.Int, len(profiles))
		for _, p := range profiles {
			name := p.Name()
			m[name] = new(expvar.Int)
			metricsMap.Set(name, m[name])
		}
		return m
	}(expvar.NewMap(metricsMapName))
)

// SyncMetrics synchronises pprof to expvar.
func SyncMetrics(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, p := range profiles {
				name := p.Name()
				metrics[name].Set(int64(p.Count()))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
