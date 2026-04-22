<div align="center">

<h1 style="border-bottom: none">
    <b><a href="https://next.yearning.io">Yearning</a></b><br />
</h1>
</div>

DBAや開発者向けに設計された、シームレスなSQL検出とクエリ監査を提供する強力なローカルデプロイプラットフォーム。プライバシーと効率に焦点を当て、MYSQL監査のための直感的で安全な環境を提供します。

---
[![OSCS Status](https://www.oscs1024.com/platform/badge/cookieY/Yearning.svg?size=small)](https://www.murphysec.com/dr/nDuoncnUbuFMdrZsh7)
![Platform Support](https://img.shields.io/badge/-x86_x64%20ARM%20Supports%20%E2%86%92-rgb(84,56,255)?style=flat-square&logoColor=white&logo=linux)
[![][github-license-shield]][github-license-link]
![GitHub top language](https://img.shields.io/github/languages/top/cookieY/Yearning?color=369eff&label=golang&labelColor=black&logo=golang&logoColor=white&style=flat-square)
[![][github-forks-shield]][github-forks-link]
[![][github-stars-shield]][github-stars-link]
[![Downloads](https://img.shields.io/github/downloads/cookieY/Yearning/total?labelColor=black&logo=download&logoColor=white&style=flat-square)](https://github.com/cookieY/Yearning/releases/latest)
[<img src="https://api.gitsponsors.com/api/badge/img?id=107417113" height="20">](https://api.gitsponsors.com/api/badge/link?p=d/F0Db/twPG+6OzJf1gggUy+f6M3CRCL2PGmI+Q/hI80KtxSQKK5f3Ec8xLanykph/60I77/SxLEkbof7Gbcetd284mOutYXz7R2BlAhNuF7GdZLrTrC7oVSTxkC9cX3WzhY1pHrAeakGpuNLaoovA==)

[English](README.md) | [简体中文](README.zh-CN.md) | 日本語

## ✨ 機能

- **AIアシスタント**: AIアシスタントはリアルタイムのSQL最適化提案を提供し、SQLのパフォーマンスを向上させます。また、テキストからSQLへの変換をサポートし、自然言語を入力して最適化されたSQL文を受け取ることができます。
  
- **SQL監査**: 承認ワークフローと自動構文チェックを備えたSQL監査チケットを作成します。SQL文の正確性、安全性、コンプライアンスを検証します。DDL/DML操作のためのロールバック文が自動生成され、トレーサビリティのための包括的な履歴ログが提供されます。

- **クエリ監査**: ユーザークエリを監査し、データソースとデータベースを制限し、機密フィールドを匿名化します。クエリ記録は将来の参照のために保存されます。

- **チェックルール**: 自動構文チェッカーは、ほとんどの自動チェックシナリオに適した広範なチェックルールをサポートします。

- **プライバシー重視**: Yearningはローカルにデプロイ可能なオープンソースソリューションであり、データベースとSQL文のセキュリティを確保します。暗号化メカニズムを含み、機密データを保護し、未承認のアクセスが発生してもデータが安全であることを保証します。

- **RBAC（ロールベースアクセス制御）**: 特定の権限を持つロールを作成および管理し、ユーザーロールに基づいてクエリワークオーダー、監査機能、およびその他の機密操作へのアクセスを制限します。

> \[!TIP]
> 詳細な情報については、[Yearningガイド](https://next.yearning.io)をご覧ください。

## ⚙️ インストール

[最新リリース](https://github.com/cookieY/Yearning/releases/latest)をダウンロードして解凍します。続行する前に、`./config.toml`が設定されていることを確認してください。

### 手動インストール

```bash
## データベースを初期化
./Yearning install

## Yearningを起動
./Yearning run

## ヘルプ
./Yearning --help
```

### 🚀 Dockerでのデプロイ
[![][docker-release-shield]][docker-release-link]
[![][docker-size-shield]][docker-size-link]
[![][docker-pulls-shield]][docker-pulls-link]
```bash
## データベースを初期化
docker run --rm -it -p8000:8000 -e SECRET_KEY=$SECRET_KEY -e MYSQL_USER=$MYSQL_USER -e MYSQL_ADDR=$MYSQL_ADDR -e MYSQL_PASSWORD=$MYSQL_PASSWORD -e MYSQL_DB=$Yearning_DB -e Y_LANG=ja_JP yeelabs/yearning "/opt/Yearning install"

## Yearningを起動
docker run -d -it -p8000:8000 -e SECRET_KEY=$SECRET_KEY -e MYSQL_USER=$MYSQL_USER -e MYSQL_ADDR=$MYSQL_ADDR -e MYSQL_PASSWORD=$MYSQL_PASSWORD -e MYSQL_DB=$Yearning_DB -e Y_LANG=ja_JP yeelabs/yearning
```
## 🤖 AIアシスタント

AIアシスタントは、大規模な言語モデルを活用してSQLの最適化提案とテキストからSQLへの変換を提供します。デフォルトまたはカスタムプロンプトを使用して、AIアシスタントは文を最適化し、自然言語入力をSQLクエリに変換することでSQLのパフォーマンスを向上させます。

![テキストからSQL](img/text2sql.jpg)

## 🔖 自動SQLチェッカー

自動SQLチェッカーは、事前定義されたルールと構文に基づいてSQL文を評価します。特定のコーディング標準、ベストプラクティス、およびセキュリティ要件に準拠していることを確認し、堅牢な検証層を提供します。

![SQL監査](img/audit.png)

## 💡 SQL構文ハイライトと自動補完

SQL構文ハイライトと自動補完機能でクエリの作成効率を向上させます。これらの機能は、SQLクエリの異なるコンポーネント（キーワード、テーブル名、列名、演算子など）を視覚的に区別するのに役立ち、クエリ構造の読み取りと理解を容易にします。

![SQLクエリ](img/query.png)

## ⏺️ オーダー/クエリ記録

プラットフォームは、ユーザーのオーダーおよびクエリ文の監査をサポートします。この機能により、データソース、データベース、および機密フィールドの処理を含むすべてのクエリ操作を追跡および記録し、規制に準拠し、クエリ履歴のトレーサビリティを提供します。

![オーダー/クエリ記録](img/record.png)

これらの主要機能に焦点を当てることで、Yearningはユーザーエクスペリエンスを向上させ、SQLのパフォーマンスを最適化し、データベース操作の強力なコンプライアンスとトレーサビリティを確保します。

## 🛠️ 推奨ツール

- [Spug - オープンソースの軽量自動化運用プラットフォーム](https://github.com/openspug/spug)

## ☎️ 連絡先

お問い合わせは、以下のメールアドレスまでご連絡ください：henry@yearning.io

## 📋 ライセンス

YearningはAGPLライセンスの下でライセンスされています。詳細については、[LICENSE](LICENSE)をご覧ください。

2024 © Henry Yee

---

Yearningを使用して、SQL監査と最適化のためのスムーズで安全かつ効率的なアプローチを体験してください。


[docker-pulls-link]: https://hub.docker.com/r/yeelabs/yearning
[docker-pulls-shield]: https://img.shields.io/docker/pulls/yeelabs/yearning?color=45cc11&labelColor=black&style=flat-square
[docker-release-link]: https://hub.docker.com/r/yeelabs/yearning
[docker-release-shield]: https://img.shields.io/docker/v/yeelabs/yearning?color=369eff&label=docker&labelColor=black&logo=docker&logoColor=white&style=flat-square
[docker-size-link]: https://hub.docker.com/r/yeelabs/yearning
[docker-size-shield]: https://img.shields.io/docker/image-size/yeelabs/yearning?color=369eff&labelColor=black&style=flat-square
[github-forks-shield]: https://img.shields.io/github/forks/cookieY/Yearning?color=8ae8ff&labelColor=black&style=flat-square
[github-forks-link]: https://github.com/cookieY/Yearning/network/members
[github-stars-link]: https://github.com/cookieY/Yearning/network/stargazers
[github-stars-shield]: https://img.shields.io/github/stars/cookieY/Yearning?color=ffcb47&labelColor=black&style=flat-square
[github-license-link]: https://github.com/cookieY/Yearning/blob/main/LICENSE
[github-license-shield]: https://img.shields.io/badge/AGPL%203.0-white?labelColor=black&style=flat-square
