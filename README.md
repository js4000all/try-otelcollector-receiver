# try-otelcollector-receiver
Otel CollectorのReceiverを実装してみる。

OpenTelemetry Collectorベースのカスタマイズは、既存 Collector に拡張機能を足すのではなく、
Collector framework を SDK のように使って、自分専用の監視エージェントを生成する感じ。


## コンパイルを通すまで

```bash
go mod init github.com/yourname/otel-practice
go mod tidy
go fmt ./...
go test ./...
```

## カスタムcollectorを作る
```sh
ocb --config builder-config.yaml
```

`builder-config.yaml`で指定した出力先に、カスタムcollectorのバイナリができる。

## 動かす

```sh
./dist/otelcol-practice --config collector-config.yaml
```

## VictoriaMetricsに送信するようにする

`builder-config.yaml`でexporterを追加する。

```yaml:builder-config.yaml
exporters:
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.158.0
  - gomod: go.opentelemetry.io/collector/exporter/otlphttpexporter v0.158.0
```

`collector-config.yaml`に`otlphttpexporter`用の設定を追加する。

```yaml:collector-config.yaml
exporters:
  debug:
    verbosity: normal
    use_internal_logger: false
    output_paths:
      - stdout
  otlphttp/vmetrics:
    endpoint: http://vm:8428/opentelemetry

service:
  pipelines:
    metrics:
      receivers: [practice]
      exporters:
        - debug
        - otlphttp/vmetrics
```

値を取得してみる。
```sh
curl 'vm:8428/api/v1/export' --data-urlencode 'match[]=practice.value' --data-urlencode 'start=-10m'
```

## ラベルを付与する

```go
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
```

例えば、こういうシリーズを登録してみる。

```go

  // temperature_celsius
	temp := sm.Metrics().AppendEmpty()
	temp.SetName("temperature_celsius")
	temp.SetDescription("Dummy temperature")
	temp.SetUnit("Cel")

	tempGauge := temp.SetEmptyGauge()

	addGaugePoint(
		tempGauge, now, dummyValue(100, counter),
		"site-a", "plc-01", "room-1",
	)

	addGaugePoint(
		tempGauge, now, dummyValue(200, counter),
		"site-a", "plc-01", "room-2",
	)

	addGaugePoint(
		tempGauge, now, dummyValue(300, counter),
		"site-b", "plc-02", "room-1",
	)

```

`temperature_celsius`を引いてみる。
```sh
curl -sG 'http://vm:8428/api/v1/export' \
  --data-urlencode 'match[]=temperature_celsius' \
  --data-urlencode 'start=-5m' \
  --data-urlencode 'end=now' | jq
```
```sh
curl -sG 'http://vm:8428/api/v1/export' \
  --data-urlencode 'match[]=temperature_celsius{site_id="site-a"}' \
  --data-urlencode 'start=-5m' \
  --data-urlencode 'end=now' | jq
```
```sh
curl -sG 'http://vm:8428/api/v1/export' \
  --data-urlencode 'match[]=temperature_celsius{measurement_position="room-2"}' \
  --data-urlencode 'start=-5m' \
  --data-urlencode 'end=now' | jq
```
```sh
curl -sG 'http://vm:8428/api/v1/export' \
  --data-urlencode 'match[]=temperature_celsius{site_id="site-a",measurement_position="room-2"}' \
  --data-urlencode 'start=-5m' \
  --data-urlencode 'end=now' | jq
```
