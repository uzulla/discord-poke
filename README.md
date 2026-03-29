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

### channel に投稿

```bash
go run . \
  --target discord-channel:1234567890 \
  --message "hello"
```

### thread に投稿

```bash
go run . \
  --target discord-thread:1487549877373239587 \
  --message "進捗どうですか？" \
  --sender-name supervisor
```

### dry-run

```bash
go run . \
  --target discord-thread:1487549877373239587 \
  --message "進捗どうですか？" \
  --dry-run
```

## target の扱い

- `discord-thread:<id>`
  - webhook URL に `?thread_id=<id>&wait=true` を付けて投稿します
- `discord-channel:<id>`
  - webhook 自体の送信先 channel に投稿します
  - MVP では channel id の照合まではしません

## 備考

- Webhook 方式なので、**新規 thread 作成**はしません
- MVP はテキスト投稿のみです
- 将来的に retry / allowlist / 重複投稿防止を追加できます
