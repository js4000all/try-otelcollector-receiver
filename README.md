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
