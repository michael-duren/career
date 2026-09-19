package database

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestRegisterMetrics(t *testing.T) {
	// sql.Open never dials, so pool stats are readable without PostgreSQL.
	s, err := Open("postgres://nobody@127.0.0.1:1/none?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	reader := sdkmetric.NewManualReader()
	reg, err := s.RegisterMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Unregister()

	var rm metricdata.ResourceMetrics
	if err = reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	points := map[string]int{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch d := m.Data.(type) {
			case metricdata.Sum[int64]:
				points[m.Name] = len(d.DataPoints)
				if m.Name == "db.client.connection.max" && d.DataPoints[0].Value != 10 {
					t.Errorf("max connections = %d, want 10", d.DataPoints[0].Value)
				}
			case metricdata.Sum[float64]:
				points[m.Name] = len(d.DataPoints)
			}
		}
	}
	want := map[string]int{
		"db.client.connection.count":         2, // idle, used
		"db.client.connection.max":           1,
		"db.client.connection.waits":         1,
		"db.client.connection.wait_duration": 1,
		"db.client.connection.closed":        3, // max_idle, max_idle_time, max_lifetime
	}
	for name, n := range want {
		if points[name] != n {
			t.Errorf("%s: %d data points, want %d", name, points[name], n)
		}
	}
}
