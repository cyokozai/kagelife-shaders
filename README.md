# kagelife-shaders

[kagelife](https://github.com/cyokozai/kagelife) で読み込む Kage（Ebitengine のシェーダー言語）のプリセット集です。

## 置き方

1 フォルダ = 1 プリセットです。kagelife はフォルダを再帰せずに読むので、`.kage` はプリセットのフォルダの直下に置きます。

```
presets/
  one-slit/
    one_slit.kage   単スリットを通る波（cnzn0「one-slit」のアレンジ）
    NOTICE          原作の MIT の著作権表示と許諾文
  basic-vj/
    01_aurora.kage … 09_glitch_strobe.kage   DJ の盛り上がりに合わせた基本の 9 枚
go.mod
shaders_test.go     presets/*/*.kage をすべて解析・型検査する検査
```

## kagelife で動かす

```sh
kagelife -shaders presets/one-slit
```

```sh
kagelife -shaders presets/basic-vj
```

`-shaders` は kagelife の `dev` ブランチにだけあり、まだリリースには入っていません。`dev` からビルドした kagelife で使ってください。

## basic-vj の 9 枚

kagelife はフォルダの `.kage` を名前の順に数字キー 1〜9 へ割り当てます。basic-vj は、番号が大きいほど激しくなるように並べています（Shift + 数字でクロスフェード）。

| キー | ファイル | 絵 | 向いている場面 |
|---|---|---|---|
| 1 | `01_aurora.kage` | オーロラ | イントロ・ブレイク（いちばん静か） |
| 2 | `02_metaballs.kage` | メタボール | 中低域のグルーヴ |
| 3 | `03_mandala.kage` | 万華鏡マンダラ | 拍ごとに一段ずつ回る線画 |
| 4 | `04_warp.kage` | ワープ | ビルドアップ |
| 5 | `05_neon_grid.kage` | ネオングリッド | 透視の床と沈む太陽 |
| 6 | `06_equalizer.kage` | イコライザ | 拍頭で跳ねる LED の棒 |
| 7 | `07_tunnel.kage` | トンネル | 拍ごとに押し寄せる輪 |
| 8 | `08_laser.kage` | レーザー | ピーク |
| 9 | `09_glitch_strobe.kage` | グリッチストロボ | ドロップ（いちばん激しい） |

9 枚に共通する決まり:

- 画面全体を光らせるのは 1 拍に 1 回までにします（光に敏感な人への配慮）。1 秒に 3 回までに収まるのは BPM が 180 以下のときだけです。kagelife の BPM は 300 まで上がり（`internal/tempo/tapper.go` の `MaxBPM`）、シェーダには BPM が渡らないので、シェーダの側では止められません。
- 4 拍ごとの周期の演出は入れません（kagelife は拍の番号を渡さないため）。ただし `06_equalizer` だけは、120 BPM を仮定して拍ごとの種を見積もっています。テンポが 120 からずれると、一部の拍で棒の形が拍の途中で切り替わります（詳しくはファイル冒頭のコメント）。
- 下の「前提にしている Uniform」の 6 つだけを使い、1 画素あたりの繰り返しを少なくします（各ファイルの冒頭にループの回数を書いています）。

kagelife を `-llm-slots` 付きで起動すると、キー 8 と 9 は LLM の枠に予約されるので、`08_laser` と `09_glitch_strobe` はキーに載りません。

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
