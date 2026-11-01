# ADR: PLC収集におけるReceiver分割方式と動的構成管理

## Status

検討中

## Context

OpenTelemetry Collector の custom receiver を用いて、PLC から Host Link によりメトリクスを収集する。

PLC ごとに複数の収集周期が存在する。例えば、一部の値は 5 秒周期、別の値は 60 秒周期で取得する。一方、同一 PLC に対する Host Link リクエストは同時に複数実行せず、必ず直列化する必要がある。異なる PLC に対しては並列アクセスしたい。

また、将来的には PLC、デバイス番号、メトリクス名、ラベル、収集周期等の収集設定をクラウド側で管理し、Collector が動的に取得する構成も選択肢としている。

この要件に対して、Receiver の分割方法として以下の2案を検討した。

## Decision Drivers

主な判断軸は以下である。

* OpenTelemetry Collector / scraper のモデルとの整合性
* Host Link 通信処理の単純さ
* 同一 PLC へのリクエスト直列化の実装難易度
* 複数周期の表現方法
* PLC 構成変更への動的な追従性
* 収集設定の管理方法
* Collector 設定ファイルの責務
* 運用時の設定変更・配布負荷
* 実装時点での技術的不確実性

---

## Option A: 収集周期ごとにReceiverインスタンスを分ける

例:

```yaml
receivers:
  plc/fast:
    collection_interval: 5s

  plc/normal:
    collection_interval: 30s

  plc/slow:
    collection_interval: 5m
```

各 receiver は、自身の周期に該当する現在の収集対象を問い合わせ、対象 PLC に対して収集要求を発行する。

同一 PLC へのリクエストは、PLC 単位の Worker / serial executor により直列化する。

概念構造は以下となる。

```text
plc/5s receiver  ──────┐
plc/30s receiver ──────┼── PLCRuntime
plc/5m receiver  ──────┘      │
                              ├── TargetRegistry
                              └── WorkerManager
                                    │
                      ┌─────────────┼─────────────┐
                      ↓             ↓             ↓
                   PLC-A          PLC-B         PLC-C
                   worker         worker        worker
```

### 利点

OpenTelemetry の scraper は `collection_interval` に従って周期的に値を取得する仕組みであるため、収集周期ごとに scraper / receiver を分ける構造はモデル上素直である。

各 receiver の責務も明確になる。

```text
receiver:
    いつ収集するか

TargetRegistry:
    現在何を収集するか

WorkerManager:
    どのPLCに要求を送るか

PLC Worker:
    同一PLCへの要求を直列化する

Host Link:
    PLCとどのように通信するか
```

PLC 一覧や収集対象を receiver の静的設定に埋め込む必要がないため、PLC の追加・削除やデバイス構成変更に動的に対応しやすい。

収集設定をクラウド API から取得する場合も、receiver 自体を再生成する必要はない。receiver は毎回現在の TargetRegistry を参照すればよい。

例えば 5 秒周期 receiver は、

```text
Scrape()
  ↓
現在の5秒周期対象を問い合わせ
  ↓
得られた対象について収集
```

という動作を継続する。

TargetRegistry の内容が実行中に変更されても、次回 Scrape から新しい構成を利用できる。

### 共有Runtime

複数 receiver から共有する `WorkerManager` や `TargetRegistry` は、OpenTelemetry Collector の extension として配置できる。

receiver は `component.Host` を通じて共有 extension を取得できるため、グローバル変数等による共有は不要である。

さらに、receiver に `TargetRegistry` と `WorkerManager` を個別に見せるのではなく、両者を内部に持つ一つの Runtime abstraction を用意する案が望ましい。

```go
type PLCRuntime interface {
    Targets(period time.Duration) []Target
    Execute(ctx context.Context, target Target) (Result, error)
}
```

receiver は、Runtime 内部で TargetRegistry と WorkerManager がどのように整合性を保っているかを知る必要がない。

### 動的構成変更

PLC 構成更新時は、TargetRegistry だけでなく WorkerManager も同時に更新する必要がある。

例えば PLC-D が新規追加された場合、

```text
新しいPLC設定を取得
    ↓
PLC-D用workerを準備
    ↓
新しいTargetRegistryを構築
    ↓
整合性を確認
    ↓
新しいRuntime stateを公開
```

という順序にする。

receiver からは常に整合済みの runtime state のみを参照可能とする。

更新途中の、

```text
TargetRegistryにはPLC-Dが存在する
WorkerManagerにはPLC-Dがまだ存在しない
```

という状態を観測させないことが重要である。

可能であれば immutable snapshot を生成し、一括で切り替える。

```text
receiver A ─── snapshot v17 ───────────→
                         |
                    state swap
                         |
receiver B ─── snapshot v18 ───────────→
```

Scrape 開始時に取得した snapshot は、その Scrape 終了まで利用する。

### APIによる設定取得

API から設定を取得する処理も、共有 Runtime または extension 側で実行可能である。

```text
ConfigUpdater
    ↓
Cloud API
    ↓
validate
    ↓
PLCRuntime.Apply(newConfig)
```

Collector component の `Start()` から background goroutine を開始し、`Shutdown()` で停止する構成が考えられる。

API 通信自体の実装難易度は高くない。

難所は以下の運用仕様である。

* 初回取得失敗時の動作
* API 障害時に前回設定を維持するか
* 空の設定を全削除と解釈するか
* 不正設定をどのように扱うか
* version 管理
* 更新途中の整合性
* PLC 削除時に処理中の worker をどう drain するか

### 主な技術的難所

A案において最も難しい部分は API 取得ではなく、PLC 単位の要求直列化である。

単純な無制限 FIFO queue とすると、PLC 障害時に古い要求が蓄積する可能性がある。

```text
5秒周期要求
    ↓ queue

30秒周期要求
    ↓ queue

PLC timeout

さらに5秒周期要求
    ↓ queue
```

監視用途では、60秒前の収集要求を後から実行する価値は低い可能性がある。

そのため Worker は一般的な worker pool よりも、

> PLC単位の serial executor

として設計するのが適切である。

検討事項として、

* pending request 数を制限する
* 古い同種 request を drop / replace する
* Scrape context が失効した request を実行しない
* PLC 障害中に backlog を無制限に増やさない

等がある。

### 評価

設計としては非常に素直であり、収集設定の動的変更にも強い。

一方、B案よりも concurrency、queue、runtime state 管理の実装が増える。

---

## Option B: PLCごとにReceiverインスタンスを分ける

PLC 1 台につき receiver / scraper instance を1つ作る。

```text
receiver/plc-a → PLC-A
receiver/plc-b → PLC-B
receiver/plc-c → PLC-C
```

同一 PLC への通信は一つの scraper 内で逐次実行する。

そのため Worker、queue、mutex 等を設けなくても、

```text
1 PLC
=
1 receiver
=
1 scrape実行主体
=
1 Host Link逐次通信主体
```

という構造で同一 PLC への並列アクセスを防止できる。

### 複数周期の表現

scraper の `collection_interval` を実際の収集周期ではなく、PLC の基本 scan period と解釈する。

例えば、

```yaml
scan_interval: 5s
```

とし、各 measurement は、

```yaml
measurements:
  - device: D100
    interval: 5s

  - device: D200
    interval: 20s

  - device: D300
    interval: 300s
```

のように指定する。

内部では、

```text
measurement interval / scan interval
```

から scan 回数を計算し、例えば 4 回に1回、60回に1回といった頻度で取得する。

内部表現として乗数を使用しても、設定には `x4`、`x60` ではなく実際の interval を記述する方が望ましい。

### 利点

プログラム構造が非常に単純になる。

```text
Scrape()
   ↓
今回のscanで読む項目を決定
   ↓
request 1
   ↓
response 1
   ↓
request 2
   ↓
response 2
```

並行処理や queue 管理をほぼ不要にできる。

また、以下の境界がすべて PLC に揃う。

```text
PLC = concurrency boundary
PLC = connection boundary
PLC = Resource boundary
PLC = receiver instance boundary
```

ResourceMetrics の構造とも相性がよい。

PLC を OpenTelemetry Resource として表現する場合、

```text
receiver instance
    ↓
PLC
    ↓
Resource
    ↓
ResourceMetrics
```

がほぼ一対一になる。

実装可能性も明確であり、現在の知識レベルでもすぐ実装可能である。

### 欠点

PLC が receiver の静的設定になるため、PLC 構成変更が Collector config の変更になる。

さらに `collector-config.yaml` に PLC を書くと、自然に以下も同じ場所へ書きたくなる。

* PLC IP address
* デバイス番号
* メトリクス名
* 収集周期
* ラベル
* measurement position 等の属性

例:

```yaml
receivers:
  plc/plc-a:
    endpoint: 192.168.1.10
    scan_interval: 5s

    measurements:
      - device: D100
        metric: temperature
        interval: 5s
        labels:
          position: inlet
```

この結果、`collector-config.yaml` が Collector pipeline の設定だけでなく、サイト固有の収集マスターを兼ねる。

これは、

```text
receiverをどう動かすか
processorをどう通すか
exporterをどこへ向けるか
```

という Collector runtime configuration と、

```text
このPLCのD100は何を意味するか
何秒周期で読むか
どの属性を付けるか
```

という設備・業務ドメイン情報が混在することを意味する。

### 設定ファイル分割案について

Collector config の純度を保つため、

```text
collector-config.yaml
site-collection.yaml
```

のように収集設定を別ファイルへ分ける案も検討した。

この案は採用候補としては弱い。

理由は、PLC の存在そのものが二箇所へ分散する可能性が高いためである。

```text
collector-config.yaml:
    receiver/plc-a が存在

site-collection.yaml:
    plc-a の詳細定義が存在
```

この場合、

* receiver はあるが PLC 定義がない
* PLC 定義はあるが receiver がない
* ID が一致しない
* 片方だけ更新される

といった不整合状態が生じる。

したがって B案を採用する場合は、Collector config が設備マスターを兼ねることを受け入れ、一箇所を source of truth とする方が筋がよい。

### 運用面

Collector は custom components を含めても single binary として配布可能であり、Ansible 等で、

```text
binary
collector-config.yaml
systemd unit
```

を配布する運用は比較的単純である。

PLC 構成変更が頻繁でなければ、

```text
Gitで設定変更
    ↓
レビュー
    ↓
Ansible適用
    ↓
Collector再起動
```

という運用でも十分成立する。

Git + Ansible には、

* 変更履歴
* レビュー
* 差分確認
* 再現性
* rollback

という利点もある。

このため、動的 API 取得より人間による設定配布を選ぶこと自体が必ずしも高コストとは限らない。

---

## A案とB案の本質的な違い

両案の違いは単なるコード構造ではない。

本質的には、

> サイトごとのPLC・収集設定を「デプロイ対象」とみなすか、「ランタイムデータ」とみなすか

という運用アーキテクチャ上の判断である。

### A案

```text
collector-config.yaml
    =
処理構造

PLC / measurement configuration
    =
runtime data
```

Collector 自体はサイト構成を静的には知らない。

現在の設備構成を Runtime / Registry から取得して動作する。

設定変更への追従責任は、主にプログラム側が負う。

```text
人間の配布作業 ↓
プログラム複雑性 ↑
```

### B案

```text
collector-config.yaml
    =
そのサイトの完全な実行定義
```

PLC 構成も Collector deployment configuration の一部である。

設定変更への追従責任は、Git、Ansible 等の構成管理・デプロイ側が負う。

```text
プログラム複雑性 ↓
構成管理側の責任 ↑
```

---

## Comparison

| 観点                  | A: 周期単位Receiver     | B: PLC単位Receiver   |
| ------------------- | ------------------- | ------------------ |
| scraperモデルとの整合      | 高い                  | scan period として再解釈   |
| 同一PLC直列化            | serial executor が必要 | 構造上自然に成立           |
| 複数PLC並列化            | ディスパッチ必要            | receiver単位で自然に成立         |
| 複数周期                | そのまま表現              | scan + interval/乗数 |
| PLC追加・削除            | 動的対応しやすい            | config変更が必要        |
| API設定取得             | 相性が良い               | receiver再構成が課題     |
| collector-configの純度 | 高い                  | 複数の関心が混在            |
| runtime実装量          | 多い                  | 少ない                |
| 運用側の配布責任            | 小さい                 | 大きい                |
| 現在の実装確実性            | 技術的に可能                 | 容易                 |
| 将来の柔軟性              | 高い                  | 中程度                |

---

## Current Assessment

B案は技術的不確実性がほぼなく、現在の知識でも即座に実装可能である。

実装構造も非常に単純であり、

```text
1 PLC = 1 receiver = 1 sequential execution unit
```

という強い不変条件を持てる。

一方、A案は設計としてより素直であり、特に PLC・measurement 構成を将来的にクラウド側から動的配信する構成との整合性が高い。

調査の結果、A案で懸念していた以下の要素はいずれも実装可能と判断した。

* 複数 receiver から共有する WorkerManager
* 共有 TargetRegistry
* runtime 中の TargetRegistry 更新
* PLC 増減に応じた WorkerManager 更新
* API からの設定取得
* Collector extension を利用した共有 Runtime の提供

A案の主要な未確定事項は、Collector framework 上で実現可能かどうかではなく、PLC serial executor の挙動をどこまで堅牢に設計するかである。

特に、

* PLC timeout 時の backlog
* 古い request の破棄
* queue 長制限
* context cancellation
* PLC 削除時の drain

について検証が必要である。

---

## Proposed Next Step

最終決定前に、A案の技術的不確実性を減らすための小規模 prototype を作成する。

実装範囲は以下に限定する。

```text
1. Collector extension として PLCRuntime を実装
2. 5秒 / 30秒の2 receiver instance から共有
3. 同一PLCへの要求が必ず直列化されることを確認
4. TargetRegistryを実行中に差し替える
5. PLC追加時にworkerを追加できることを確認
```

この段階では Cloud API は実装しない。

TargetRegistry 更新はテストコードまたは固定 timer から行う。

この prototype が問題なく成立すれば、A案の最大の技術的不確実性は解消される。

API polling はその後に追加する。

---

## Decision

現時点では最終決定を保留する。

B案は既知の技術で直ちに実装可能な fallback として保持する。

A案について、Collector extension + shared PLCRuntime + PLC serial executor の prototype を実施し、実装複雑性を実測した上で採否を決定する。

特にA案の評価では、「実装可能か」ではなく、

> 動的構成変更によって得られる運用上の利点が、serial executor と runtime state 管理の追加複雑性に見合うか

を最終的な判断基準とする。
