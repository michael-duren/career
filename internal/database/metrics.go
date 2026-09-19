package database

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	meterName = "github.com/michael-duren/career-strategy/internal/database"
	poolName  = "career"
)

// RegisterMetrics reports the connection pool's sql.DBStats as observable
// instruments, read once per collection. Unregister the returned
// registration before closing the Store.
func (s *Store) RegisterMetrics(mp metric.MeterProvider) (metric.Registration, error) {
	m := mp.Meter(meterName)
	count, err := m.Int64ObservableUpDownCounter("db.client.connection.count",
		metric.WithUnit("{connection}"),
		metric.WithDescription("Connections currently in the pool, by state."))
	if err != nil {
		return nil, err
	}
	maxOpen, err := m.Int64ObservableUpDownCounter("db.client.connection.max",
		metric.WithUnit("{connection}"),
		metric.WithDescription("Maximum number of open connections allowed."))
	if err != nil {
		return nil, err
	}
	waits, err := m.Int64ObservableCounter("db.client.connection.waits",
		metric.WithUnit("{wait}"),
		metric.WithDescription("Connection requests that had to wait for a free connection."))
	if err != nil {
		return nil, err
	}
	waitDuration, err := m.Float64ObservableCounter("db.client.connection.wait_duration",
		metric.WithUnit("s"),
		metric.WithDescription("Total time spent waiting for a free connection."))
	if err != nil {
		return nil, err
	}
	closed, err := m.Int64ObservableCounter("db.client.connection.closed",
		metric.WithUnit("{connection}"),
		metric.WithDescription("Connections closed by the pool, by reason."))
	if err != nil {
		return nil, err
	}

	pool := semconv.DBClientConnectionPoolName(poolName)
	var (
		all      = metric.WithAttributes(pool)
		idle     = metric.WithAttributes(pool, semconv.DBClientConnectionStateIdle)
		used     = metric.WithAttributes(pool, semconv.DBClientConnectionStateUsed)
		reason   = attribute.Key("reason")
		maxIdle  = metric.WithAttributes(pool, reason.String("max_idle"))
		idleTime = metric.WithAttributes(pool, reason.String("max_idle_time"))
		lifetime = metric.WithAttributes(pool, reason.String("max_lifetime"))
	)
	return m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		st := s.DB.Stats()
		o.ObserveInt64(count, int64(st.Idle), idle)
		o.ObserveInt64(count, int64(st.InUse), used)
		o.ObserveInt64(maxOpen, int64(st.MaxOpenConnections), all)
		o.ObserveInt64(waits, st.WaitCount, all)
		o.ObserveFloat64(waitDuration, st.WaitDuration.Seconds(), all)
		o.ObserveInt64(closed, st.MaxIdleClosed, maxIdle)
		o.ObserveInt64(closed, st.MaxIdleTimeClosed, idleTime)
		o.ObserveInt64(closed, st.MaxLifetimeClosed, lifetime)
		return nil
	}, count, maxOpen, waits, waitDuration, closed)
}
