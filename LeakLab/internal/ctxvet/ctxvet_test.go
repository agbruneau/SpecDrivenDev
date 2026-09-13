package ctxvet

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestUC003_MainFlow(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		calls []string
	}{
		{"net.Dial visé", `package p
import "net"
func f() { net.Dial("tcp", "x") }`, []string{"net.Dial"}},
		{"DialContext épargné", `package p
import ("context"; "net")
func f(ctx context.Context) { var d net.Dialer; d.DialContext(ctx, "tcp", "x") }`, nil},
		{"http.Get et NewRequest", `package p
import "net/http"
func f() { http.Get("u"); http.NewRequest("GET", "u", nil) }`, []string{"http.Get", "http.NewRequest"}},
		{"BR-003-2 alias d'import", `package p
import web "net/http"
func f() { web.Post("u", "", nil) }`, []string{"web.Post"}},
		{"BR-003-2 variable homonyme", `package p
import "net/http"
var _ = http.MethodGet
type client struct{}
func (client) Get(string) {}
func f() { http := client{}; http.Get("u") }`, nil},
		{"BR-003-3 méthodes non visées", `package p
import "net"
func f(c net.Conn) { c.Read(nil) }`, nil},
		{"exec.Command", `package p
import "os/exec"
func f() { exec.Command("go") }`, []string{"exec.Command"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeFiles(t, map[string]string{"a.go": tt.src})
			diags, err := Analyze(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(diags) != len(tt.calls) {
				t.Fatalf("diagnostics = %v, appels attendus %v", diags, tt.calls)
			}
			for i, d := range diags {
				if d.Call != tt.calls[i] || d.Replacement == "" || d.Line == 0 {
					t.Errorf("diagnostic %d = %+v, %s attendu", i, d, tt.calls[i])
				}
			}
		})
	}
}

func TestUC003_TestsExclusEtTri(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"b.go":      "package p\nimport \"net\"\nfunc g() {\n\tnet.Listen(\"tcp\", \"x\")\n\tnet.Dial(\"tcp\", \"x\")\n}\n",
		"a.go":      "package p\nimport \"net\"\nfunc f() { net.Dial(\"tcp\", \"x\") }\n",
		"a_test.go": "package p\nimport \"net\"\nfunc h() { net.Dial(\"tcp\", \"x\") }\n",
	})
	diags, err := Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 3 || filepath.Base(diags[0].File) != "a.go" || diags[1].Line != 4 || diags[2].Line != 5 {
		t.Fatalf("diagnostics = %v", diags)
	}
}

func TestUC003_A1_SyntaxeInvalide(t *testing.T) {
	dir := writeFiles(t, map[string]string{"a.go": "package p\nfunc {"})
	if diags, err := Analyze(dir); err == nil || diags != nil {
		t.Fatalf("Analyze = %v, %v ; erreur sans diagnostic attendue", diags, err)
	}
}
