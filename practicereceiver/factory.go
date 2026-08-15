package practicereceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

var typeID = component.MustNewType("practice")

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeID,
		createDefaultConfig,
		receiver.WithMetrics(
			createMetricsReceiver,
			component.StabilityLevelDevelopment,
		),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		ControllerConfig: scraperhelper.NewDefaultControllerConfig(),
	}
}

func createMetricsReceiver(
	ctx context.Context,
	settings receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	config := cfg.(*Config)

	s := newScraper()

	sc, err := scraper.NewMetrics(s.scrape)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewMetricsController(
		&config.ControllerConfig,
		settings,
		nextConsumer,
		scraperhelper.AddMetricsScraper(typeID, sc),
	)
}
