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
