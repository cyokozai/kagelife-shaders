# kagelife-shaders

[kagelife](https://github.com/cyokozai/kagelife) で読み込む Kage（Ebitengine のシェーダー言語）のプリセット集です。

## 置き方

1 フォルダ = 1 プリセットです。kagelife はフォルダを再帰せずに読むので、`.kage` はプリセットのフォルダの直下に置きます。

```
presets/
  one-slit/
    one_slit.kage   単スリットを通る波（cnzn0「one-slit」のアレンジ）
    NOTICE          原作の MIT の著作権表示と許諾文
go.mod
shaders_test.go     presets/*/*.kage をすべて解析・型検査する検査
```

## kagelife で動かす

```sh
kagelife -shaders presets/one-slit
```

`-shaders` は kagelife の `dev` ブランチにだけあり、まだリリースには入っていません。`dev` からビルドした kagelife で使ってください。

## 前提にしている Uniform

kagelife（`dev`）の `internal/uniform/uniform.go` が毎フレーム渡す次の 6 つだけを使います。使わないものは宣言しなくてかまいません。

| 名前 | Kage の型 | 中身 |
|---|---|---|
| `Time` | `float` | 起動からの秒数 |
| `Beat` | `float` | 拍の位相（0〜1） |
| `Frame` | `int` | フレーム番号 |
| `Random` | `float` | 0〜1 の乱数（毎フレーム変わる） |
| `Resolution` | `vec2` | 描画先の画素数 |
| `Cursor` | `vec2` | カーソルの位置（画素） |

どのシェーダーも 1 行目（先頭 5 行以内）に `//kage:unit pixels` を書きます。

## 検査

`shaders_test.go` が、すべての `presets/*/*.kage` について次を確かめます。CI（`.github/workflows/ci.yml`）も同じものを Linux で走らせます。

- `ebiten.NewShader` で Kage として解析・型検査できる（GLSL・MSL・HLSL への変換と GPU ドライバでのコンパイルは最初の描画のときに行われるので、この検査では確かめていません）
- 先頭 5 行以内に `//kage:unit pixels` がある
- 宣言した Uniform が上の 6 つの名前と型に収まる

cgo と X11・OpenGL の開発パッケージが要るので、手元ではコンテナの中で走らせます（例: `golang:1.27.3` に CI と同じ apt のパッケージを入れた image で `go test ./...`）。

## ライセンス

このリポジトリにはライセンスを付けていません（所有者の判断です）。

ただし `presets/one-slit/` は cnzn0 の [one-slit](https://github.com/cnzn0/one-slit)（MIT License）を元にしたアレンジなので、原作の著作権表示と許諾文を `presets/one-slit/NOTICE` に置いています。
