# discord-poke

OpenClaw cron などから単発で呼びやすい、Discord 投稿用の小さな CLI です。

MVP では **Discord Webhook** を使って、既存 channel / existing thread にテキスト投稿します。

## できること

- `discord-channel:<id>` / `discord-thread:<id>` 形式の target を受ける
- メッセージを 1 回投稿する
- `--sender-name` で webhook username を上書きする
- `--dry-run` で送信せず確認する
- 成功時に `messageId` / `timestamp` / `target` を JSON で出力する
- 失敗時に stderr と非0終了コードを返す

## 必要な環境変数

```bash
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
```

## 使い方

`-h` / `--help` では、必須環境変数 `DISCORD_WEBHOOK_URL` と target の意味、実行例も表示されます。

### ローカルでビルドして使う

```bash
go build -o discord-poke .
```

### channel を検証して投稿

```bash
./discord-poke \
  --target discord-channel:1485530659924611102 \
  --message "hello"
```

### thread に投稿

```bash
./discord-poke \
  --target discord-thread:1487549877373239587 \
  --message "進捗どうですか？" \
  --sender-name supervisor
```

### dry-run

```bash
./discord-poke \
  --target discord-thread:1487549877373239587 \
  --message "進捗どうですか？" \
  --dry-run
```

## target の扱い

- `discord-thread:<id>`
  - webhook URL に `?thread_id=<id>&wait=true` を付けて投稿します
  - こちらは配送先 thread の指定として使われます

- `discord-channel:<id>`
  - **配送先 channel を切り替える指定ではありません**
  - webhook が元々紐づいている channel が、指定した `<id>` と一致するかを検証するための指定です
  - 一致しない場合は **失敗扱い** になり、投稿しません

## Release builds

GitHub Release を publish すると、GitHub Actions で次のバイナリをビルドして release asset に添付します。

- `discord-poke-linux-amd64`
- `discord-poke-linux-amd64.sha256`
- `discord-poke-darwin-arm64`
- `discord-poke-darwin-arm64.sha256`

トリガー:
- GitHub Release の `published`

workflow:
- `.github/workflows/release.yml`

## 備考

- Webhook 方式なので、**新規 thread 作成**はしません
- MVP はテキスト投稿のみです
- 将来的に retry / allowlist / 重複投稿防止を追加できます
