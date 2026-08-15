package practicereceiver

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type practiceScraper struct {
	counter int64
}

func newScraper() *practiceScraper {
	return &practiceScraper{}
}

func (s *practiceScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	n := atomic.AddInt64(&s.counter, 1)
	counter := n % 100

	md := pmetric.NewMetrics()

	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()

	now := pcommon.NewTimestampFromTime(time.Now())

	// temperature_celsius
	temp := sm.Metrics().AppendEmpty()
	temp.SetName("temperature_celsius")
	temp.SetDescription("Dummy temperature")
	temp.SetUnit("Cel")

	tempGauge := temp.SetEmptyGauge()

	addGaugePoint(
		tempGauge,
		now,
		dummyValue(100, counter),
		"site-a",
		"plc-01",
		"room-1",
	)

	addGaugePoint(
		tempGauge,
		now,
		dummyValue(200, counter),
		"site-a",
		"plc-01",
		"room-2",
	)

	addGaugePoint(
		tempGauge,
		now,
		dummyValue(300, counter),
		"site-b",
		"plc-02",
		"room-1",
	)

	// humidity_percent
	humidity := sm.Metrics().AppendEmpty()
	humidity.SetName("humidity_percent")
	humidity.SetDescription("Dummy humidity")
	humidity.SetUnit("%")

	humidityGauge := humidity.SetEmptyGauge()

	addGaugePoint(
		humidityGauge,
		now,
		dummyValue(400, counter),
		"site-a",
		"plc-01",
		"room-1",
	)

	return md, nil
}

func dummyValue(base int, counter int64) float64 {
	upper := base + rand.Intn(100)

	return float64(upper*100 + int(counter))
}

func addGaugePoint(
	gauge pmetric.Gauge,
	timestamp pcommon.Timestamp,
	value float64,
	siteID string,
	targetID string,
	position string,
) {
	dp := gauge.DataPoints().AppendEmpty()

	dp.SetTimestamp(timestamp)
	dp.SetDoubleValue(value)

	attrs := dp.Attributes()
	attrs.PutStr("site_id", siteID)
	attrs.PutStr("target_id", targetID)
	attrs.PutStr("measurement_position", position)
}
