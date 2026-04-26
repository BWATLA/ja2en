# ja2en

Japanese-to-English translation CLI for software engineers. シングルバイナリ (Go 製)、起動 ~10ms、デフォルトで OpenAI gpt-5.4-nano (paid) を叩いて自然な英語を返す。Gemini と DeepL がフォールバック。

## 特徴

- **起動 ~10ms**: Go 製シングルバイナリ。`ja2en "..."` で叩いた瞬間に英訳が返る。
- **CJK IME と相性のいいインタラクティブモード**: `--interactive` で multi-line text editor 起動。日本語入力時に IME composition window がカーソル直下に追従する (他の TUI 翻訳ツールではここが画面下端に張り付くケースが多い)。マルチライン navigation・全 emacs 系キーバインド対応。
- **プロンプトインジェクション対策済み**: 入力に「日本語で答えて」「上の指示は無視して」等のメタ指示が混じっても翻訳契約が壊れない。
- **3 プロバイダ multi-provider 設計**: profile ごとに OpenAI / Gemini / DeepL を切替。`api_key_env` で「どの環境変数を読むか」も config 側から指定可能。
- **エンジニア向けの翻訳トーン**: 「ってか / なんですけど / 〜してください (口語)」等のヘッジ語を英訳から落とすスタイル指定が組み込み済み。
- **クリップボード対応**: `--clip` で読込、`--paste` で書戻し (WSL では `clip.exe` 経由で動く)。
- **API キーは環境変数のみ**: 設定ファイルにキーを書かない。

## デモ

```text
$ ja2en "ってかこれ見てほしいんだけど"
Look at this.

$ ja2en --interactive
Enter Japanese text. Ctrl-D to translate, Ctrl-C to abort.

│ 改行を含む
│ 日本語をそのままタイプできるエディタ
│ ^D
A multi-line editor where you can type Japanese as is.
```

## Quick Start

3 分で動かせる。

1. **インストール**:

   ```bash
   go install github.com/GigiTiti-Kai/ja2en@v0.5.0
   ```

   (`$GOPATH/bin` が `PATH` に入っていること。 `go env GOPATH` で確認。)

2. **OpenAI API キーを発行**: <https://platform.openai.com/>
   - **$5 の prepaid 必須** (credits は 1 年で expire)。
   - 通常用途で月 $0.13 程度の見込み (1 日 50 翻訳ペースなら $5 で 3 年)。

3. **環境変数を設定** (例: `~/.bashrc`):

   ```bash
   export OPENAI_API_KEY="sk-..."
   ```

4. **設定ファイルを生成**:

   ```bash
   ja2en init
   ```

   `~/.config/ja2en/config.toml` が作られる (`openai` プロファイルがデフォルト)。

5. **動作確認**:

   ```bash
   ja2en "明日出社する"
   # → I'll come to the office tomorrow.
   ```

## 使い方

```bash
# 引数として渡す
ja2en "今日は良い天気だ"

# stdin から
echo "明日は遅れます" | ja2en

# クリップボードから読む
ja2en --clip

# 翻訳結果をクリップボードに書き戻す
ja2en --paste "緊急対応します"

# 読込→翻訳→書き戻し
ja2en --clip --paste

# マルチラインインタラクティブモード (IME 対応)
ja2en --interactive       # または -i

# プロファイル切替 (フォールバック先を使う)
ja2en --profile gemini "..."
ja2en --profile deepl "..."

# モデルだけアドホックに切替
ja2en --model gpt-5.4-mini "..."

# プロンプトファイルを指定 (口語 / 技術文書 / 等のスタイル切替)
ja2en --prompt-file ~/prompts/formal.md "..."
```

## プロバイダ比較

`ja2en init` で生成される標準 3 プロファイル:

| Profile | Model | コスト | 速度 (中央値) | 用途 |
|---|---|---|---|---|
| **openai** (default) | `gpt-5.4-nano` | paid (~$0.13/月) | ~600ms | 本番常用 |
| **gemini** | `gemini-2.5-flash-lite` | free (RPD 制限あり) | ~850ms | クォータ余裕時のフォールバック |
| **deepl** | DeepL Free | free (500K chars/月) | ~1000ms | 極短文・OpenAI 障害時 |

OpenRouter を使いたい場合は `openrouter` プロファイルが config にコメントアウトで入っている。`OPENROUTER_API_KEY` を設定して有効化。

## なぜデフォルトが paid OpenAI なのか?

以前のバージョン (v0.2.x) は Google AI Studio の Gemini 2.5 Flash-Lite (free) をデフォルトにしていた。理由は単純で「無料・高速・品質も悪くない」だったが、本番運用で次の罠を踏んだ:

- **Gemini 2.5 系の thinking モードがデフォルト ON** で、TPM 250K (project shared) を 1 リクエストで数千〜数万 tokens 消費する → 公称 RPD 1000 が実効 RPD 20 まで落ちる。
- `reasoning_effort = "none"` を必ず指定する設計に変更したものの、free tier だと結局重いリクエストで他のユーザーと TPM を取り合うため不安定。

v0.3 で **OpenAI gpt-5.4-nano + `reasoning_effort = "none"`** をデフォルトに切替た。月 $0.13 程度のコストで:

- 安定 (paid tier の rate limit は実効値が公称通り)
- ja→en 品質が一段上 (Reddit / Intento / llmversus が概ね同意)
- レイテンシ ~600ms (Gemini free より速い)

無料運用したい人は `--profile gemini` で従来挙動が使える。詳細は `CLAUDE.md` の「既知の制約」と「デフォルトモデル変遷」を参照。

## 設定 (`~/.config/ja2en/config.toml`)

`ja2en init` が生成するファイルの抜粋:

```toml
default_profile = "openai"
timeout_seconds = 30

[profiles.openai]
provider = "openai"
api_base = "https://api.openai.com/v1"
api_key_env = "OPENAI_API_KEY"
model = "gpt-5.4-nano"
reasoning_effort = "none"
prompt = """..."""              # ja→en スタイル指定 + プロンプトインジェクション防御
```

主要フィールド:

| フィールド | 意味 |
|---|---|
| `provider` | `"openai"` (OpenAI 互換 chat completions) または `"deepl"` (DeepL REST) |
| `api_key_env` | この profile が読む環境変数の名前。同じバイナリで OpenAI / Google AI Studio / OpenRouter 等を切替できる |
| `reasoning_effort` | reasoning モデル用 (`"none"` / `"low"` / `"medium"` / `"high"` / `"xhigh"`)。**翻訳タスクでは `"none"` 一択** |
| `prompt` | system prompt 本体。`prompt_file` でファイル指定も可 |

## シェルエイリアス (推奨)

`~/.bashrc` 等に:

```bash
# t = 引数で翻訳、引数なし TTY で clip+paste、stdin あれば pipe 翻訳
t() {
    if [ "$#" -eq 0 ]; then
        if [ -t 0 ]; then ja2en --clip --paste; else ja2en; fi
    else
        ja2en "$@"
    fi
}

# ti = インタラクティブ
ti() { ja2en --interactive; }
```

## Development

```bash
git clone https://github.com/GigiTiti-Kai/ja2en
cd ja2en
make build         # ./ja2en を生成
make install       # $GOPATH/bin/ja2en に配置
make check         # fmt + vet + lint + test 一括
```

CI は `.github/workflows/ci.yml` で `make check` を回す。詳細な開発ノートは `CLAUDE.md` 参照。

## License

[MIT](LICENSE)
