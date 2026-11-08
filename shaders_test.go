package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// unitDirective は全シェーダに必須のディレクティブ（kagelife dev の shaders_test.go と同じ）。
// Ebitengine v2.10 系は texels 既定だとコンパイルエラーの行番号がずれるため、pixels を明示させる。
const unitDirective = "//kage:unit pixels"

// unitDirectiveMaxLine はディレクティブを探す先頭からの行数。
const unitDirectiveMaxLine = 5

// knownUniforms は kagelife（dev）の internal/uniform/uniform.go が毎フレーム渡す Uniform と、その Kage での型。
// これ以外の Uniform を宣言したシェーダは kagelife では値が入らないので、テストで止める。
var knownUniforms = map[string]string{
	"Time":       "float",
	"Beat":       "float",
	"Frame":      "int",
	"Random":     "float",
	"Resolution": "vec2",
	"Cursor":     "vec2",
}

// shaderFiles は presets/<プリセット>/*.kage をすべて返す。1 フォルダ = 1 プリセット。
func shaderFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("presets", "*", "*.kage"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("presets/*/*.kage が 1 件も見つからない")
	}
	return files
}

func readShader(t *testing.T, path string) []byte {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return src
}

// TestShadersParseAsKage は全シェーダが Kage として解析・型検査できることを確かめる。
// NewShader は Kage を中間表現にするところまでを同期で行い、エラーを返す。GLSL・MSL・HLSL への変換と
// GPU ドライバでのコンパイルは最初の描画のときに走るので、この検査では確かめていない。
// NewShader は RunGame 前・DISPLAY 無しでも呼べる。
func TestShadersParseAsKage(t *testing.T) {
	for _, path := range shaderFiles(t) {
		t.Run(path, func(t *testing.T) {
			s, err := ebiten.NewShader(readShader(t, path))
			if err != nil {
				t.Fatalf("NewShader: %v", err)
			}
			s.Deallocate()
		})
	}
}

// TestShadersDeclarePixelUnit は全シェーダの先頭付近に //kage:unit pixels があることを確かめる。
func TestShadersDeclarePixelUnit(t *testing.T) {
	for _, path := range shaderFiles(t) {
		t.Run(path, func(t *testing.T) {
			sc := bufio.NewScanner(bytes.NewReader(readShader(t, path)))
			for i := 0; i < unitDirectiveMaxLine && sc.Scan(); i++ {
				if strings.TrimSpace(sc.Text()) == unitDirective {
					return
				}
			}
			t.Fatalf("先頭 %d 行以内に %q が無い", unitDirectiveMaxLine, unitDirective)
		})
	}
}

// uniformProblems は src のパッケージ直下の Uniform 宣言のうち、kagelife の渡す 6 つの名前と型に
// 収まらないものを 1 件 1 行で返す。
// Kage は Go の構文で書くので go/parser で読める。パッケージ直下の var（括弧でまとめたもの・1 行に複数の名前も含む）
// をすべて Uniform として調べる。関数の中の var は対象外。
func uniformProblems(src []byte) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "shader.kage", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			typ := ""
			if vs.Type != nil {
				typ = types.ExprString(vs.Type)
			}
			for _, id := range vs.Names {
				want, ok := knownUniforms[id.Name]
				if !ok {
					problems = append(problems, fmt.Sprintf("Uniform %q は kagelife が渡さない", id.Name))
					continue
				}
				if typ != want {
					problems = append(problems, fmt.Sprintf("Uniform %q の型が %q（kagelife は %q で渡す）", id.Name, typ, want))
				}
			}
		}
	}
	return problems, nil
}

// TestUniformProblems は、Uniform の宣言の書き方が変わっても検査が見落とさないことを確かめる。
func TestUniformProblems(t *testing.T) {
	const head = "//kage:unit pixels\npackage main\n\n"
	cases := []struct {
		name string
		src  string
		want int // 見つかるべき問題の数
	}{
		{"正しい宣言", "var Time float\nvar Frame int\nvar Cursor vec2\n", 0},
		{"行末コメント付き", "var Mouse vec2 // カーソル\n", 1},
		{"1 行に複数の名前", "var Foo, Bar float\n", 2},
		{"括弧でまとめた宣言", "var (\n\tTime float\n\tMouse vec2\n\tFrame float\n)\n", 2},
		{"配列の型", "var W [4]float\n", 1},
		{"知っている名前で配列の型", "var Time [4]float\n", 1},
		{"関数の中の var は Uniform ではない", "func f() float {\n\tvar x [4]float\n\treturn x[0]\n}\n", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := uniformProblems([]byte(head + c.src))
			if err != nil {
				t.Fatalf("uniformProblems: %v", err)
			}
			if len(got) != c.want {
				t.Fatalf("問題の数 %d（%q）、期待 %d", len(got), got, c.want)
			}
		})
	}
}

// TestShadersUseKnownUniforms は、宣言された Uniform が kagelife の渡す 6 つの名前と型に収まることを確かめる。
func TestShadersUseKnownUniforms(t *testing.T) {
	for _, path := range shaderFiles(t) {
		t.Run(path, func(t *testing.T) {
			problems, err := uniformProblems(readShader(t, path))
			if err != nil {
				t.Fatalf("uniformProblems: %v", err)
			}
			for _, p := range problems {
				t.Error(p)
			}
		})
	}
}
