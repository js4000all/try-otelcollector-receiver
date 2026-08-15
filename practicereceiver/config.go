package practicereceiver

import "go.opentelemetry.io/collector/scraper/scraperhelper"

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
}
