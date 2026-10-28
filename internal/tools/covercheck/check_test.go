package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleProfile = `mode: atomic
github.com/x/m/a/a.go:1.1,2.2 4 1
github.com/x/m/a/a.go:3.1,4.2 6 0
github.com/x/m/a/mock_gen.go:1.1,2.2 50 0
github.com/x/m/b/b.go:1.1,2.2 5 3
github.com/x/m/bmock/m.go:1.1,2.2 9 0
github.com/x/m/internal/t/t.go:1.1,2.2 9 0
`

func TestParseProfile(t *testing.T) {
	got, err := parseProfile(strings.NewReader(sampleProfile))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 measured packages, got %v", got)
	}
	if got["github.com/x/m/a"] != 40 {
		t.Errorf("a = %v, want 40 (generated mock_*.go must be ignored)", got["github.com/x/m/a"])
	}
	if got["github.com/x/m/b"] != 100 {
		t.Errorf("b = %v, want 100", got["github.com/x/m/b"])
	}
}

func TestParseProfile_Malformed(t *testing.T) {
	for name, in := range map[string]string{
		"no colon":   "mode: atomic\nbroken\n",
		"few fields": "mode: atomic\nf.go:1.1,2.2 4\n",
		"bad number": "mode: atomic\nf.go:1.1,2.2 x 1\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseProfile(strings.NewReader(in)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name     string
		current  map[string]float64
		baseline map[string]float64
		want     int
	}{
		{"unchanged", map[string]float64{"p": 50}, map[string]float64{"p": 50}, 0},
		{"improved", map[string]float64{"p": 60}, map[string]float64{"p": 50}, 0},
		{"within tolerance", map[string]float64{"p": 49.6}, map[string]float64{"p": 50}, 0},
		{"dropped", map[string]float64{"p": 49}, map[string]float64{"p": 50}, 1},
		{"new package too low", map[string]float64{"n": 79.9}, map[string]float64{}, 1},
		{"new package ok", map[string]float64{"n": 80}, map[string]float64{}, 0},
		{"removed package ignored", map[string]float64{}, map[string]float64{"gone": 90}, 0},
		{"two problems", map[string]float64{"a": 1, "b": 2}, map[string]float64{"a": 50}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evaluate(tt.current, tt.baseline); len(got) != tt.want {
				t.Errorf("got %d problems %v, want %d", len(got), got, tt.want)
			}
		})
	}
}

func TestBaselineRoundTrip(t *testing.T) {
	file := t.TempDir() + "/base.txt"
	in := map[string]float64{"github.com/x/m/a": 41.25, "github.com/x/m/b": 100}
	if err := writeBaseline(file, in); err != nil {
		t.Fatal(err)
	}
	out, err := readBaseline(file)
	if err != nil {
		t.Fatal(err)
	}
	if out["github.com/x/m/a"] != 41.3 && out["github.com/x/m/a"] != 41.2 || out["github.com/x/m/b"] != 100 {
		t.Errorf("round trip mismatch: %v", out)
	}
}

func TestReadBaseline_Errors(t *testing.T) {
	if _, err := readBaseline(t.TempDir() + "/missing.txt"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMeasured(t *testing.T) {
	tests := map[string]bool{
		"github.com/x/m/cache":                true,
		"github.com/x/m/cache/cachemock":      false,
		"github.com/x/m/internal/archtest":    false,
		"github.com/x/m/queue/kafka/producer": true,
	}
	for pkg, want := range tests {
		if got := measured(pkg); got != want {
			t.Errorf("measured(%q) = %v, want %v", pkg, got, want)
		}
	}
}

func TestHasStatements(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("iface.go", "package p\n\ntype I interface{ M() }\n")
	write("empty.go", "package p\n\nfunc F() {}\n")
	write("code.go", "package p\n\nfunc G() int { return 1 }\n")
	write("mock_x.go", "package p\n\nfunc H() int { return 1 }\n")
	write("broken.go", "package p\n\nfunc (")

	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{"interfaces only", []string{"iface.go"}, false},
		{"empty function body", []string{"iface.go", "empty.go"}, false},
		{"real code", []string{"iface.go", "code.go"}, true},
		{"generated mock is ignored", []string{"mock_x.go"}, false},
		{"unparsable file is counted", []string{"broken.go"}, true},
		{"no files", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasStatements(dir, tt.files); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
