package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
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

// uniformDecl はパッケージ直下の Uniform 宣言（例: var Time float）に当たる。
var uniformDecl = regexp.MustCompile(`(?m)^var\s+(\w+)\s+(\w+)\s*$`)

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

// TestShadersCompile は全シェーダが Kage としてコンパイルできることを確かめる。
// NewShader は RunGame 前・DISPLAY 無しでも呼べる。
func TestShadersCompile(t *testing.T) {
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

// TestShadersUseKnownUniforms は、宣言された Uniform が kagelife の渡す 6 つの名前と型に収まることを確かめる。
func TestShadersUseKnownUniforms(t *testing.T) {
	for _, path := range shaderFiles(t) {
		t.Run(path, func(t *testing.T) {
			for _, m := range uniformDecl.FindAllStringSubmatch(string(readShader(t, path)), -1) {
				name, typ := m[1], m[2]
				want, ok := knownUniforms[name]
				if !ok {
					t.Errorf("Uniform %q は kagelife が渡さない", name)
					continue
				}
				if typ != want {
					t.Errorf("Uniform %q の型が %q（kagelife は %q で渡す）", name, typ, want)
				}
			}
		})
	}
}
