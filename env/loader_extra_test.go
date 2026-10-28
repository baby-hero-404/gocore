package env

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func errCode(t *testing.T, err error) stderr.Code {
	t.Helper()
	var se *stderr.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *stderr.Error, got %T (%v)", err, err)
	}
	return se.Code
}

type fileConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

func TestLoad_ConfigFileSources(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.yaml", "name: from-yaml\nport: 7\n")
	writeFile(t, dir, "other.json", `{"name":"from-json","port":9}`)
	bad := writeFile(t, dir, "bad.yaml", "name: [unclosed\n")

	tests := []struct {
		name     string
		opts     []Option
		want     fileConfig
		wantCode stderr.Code
	}{
		{"direct file", []Option{WithConfigFile(filepath.Join(dir, "app.yaml"))}, fileConfig{"from-yaml", 7}, 0},
		{"name path and type", []Option{WithConfigName("app"), WithConfigPath(dir), WithConfigType("yaml")}, fileConfig{"from-yaml", 7}, 0},
		{"json by name", []Option{WithConfigName("other"), WithConfigPath(dir), WithConfigType("json")}, fileConfig{"from-json", 9}, 0},
		{"name not found anywhere is tolerated", []Option{WithConfigName("absent"), WithConfigPath(dir), WithConfigType("yaml")}, fileConfig{}, 0},
		{"explicit missing file is an error", []Option{WithConfigFile(filepath.Join(dir, "nope.yaml"))}, fileConfig{}, stderr.CodeInvalidParam},
		{"malformed file is an error", []Option{WithConfigFile(bad)}, fileConfig{}, stderr.CodeInvalidParam},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got fileConfig
			err := Load("", &got, tt.opts...)
			if tt.wantCode != 0 {
				if errCode(t, err) != tt.wantCode {
					t.Fatalf("code = %v, want %v (err=%v)", errCode(t, err), tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoad_PrefixAndEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	file := writeFile(t, dir, "app.yaml", "name: from-file\nport: 1\n")
	t.Setenv("SVC_PORT", "4242")

	var got fileConfig
	if err := Load("svc", &got, WithConfigFile(file)); err != nil {
		t.Fatal(err)
	}
	if got.Name != "from-file" || got.Port != 4242 {
		t.Errorf("got %+v, want env to override port only", got)
	}

	var viaOption fileConfig
	if err := Load("", &viaOption, WithPrefix("svc"), WithConfigFile(file)); err != nil {
		t.Fatal(err)
	}
	if viaOption.Port != 4242 {
		t.Errorf("WithPrefix not applied: %+v", viaOption)
	}
}

func TestLoad_UnmarshalFailureIsInvalidParam(t *testing.T) {
	t.Setenv("BADT_PORT", "not-a-number")
	var got fileConfig
	if err := Load("badt", &got); errCode(t, err) != stderr.CodeInvalidParam {
		t.Fatalf("want CodeInvalidParam, got %v", err)
	}
}

func TestLoad_MissingEnvFileIsIgnored(t *testing.T) {
	var got fileConfig
	if err := Load("", &got, WithEnvFile(filepath.Join(t.TempDir(), "missing.env"))); err != nil {
		t.Fatalf("a missing .env must be ignored, got %v", err)
	}
}

func TestLoad_EnvFileParsing(t *testing.T) {
	t.Setenv("ENVF_NAME", "")
	t.Setenv("ENVF_PORT", "")
	t.Setenv("ENVF_KEEP", "from-process")
	file := writeFile(t, t.TempDir(), "x.env", "# comment\n\nENVF_NAME=\"quoted\"\nENVF_PORT = 99\nENVF_KEEP=from-file\nnot a pair\n")

	if err := loadDotEnv(file); err != nil {
		t.Fatal(err)
	}
	tests := map[string]string{"ENVF_NAME": "quoted", "ENVF_PORT": "99", "ENVF_KEEP": "from-process"}
	for k, want := range tests {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q, want %q (process env must win over file)", k, got, want)
		}
	}
}

type nested struct {
	Secret string `mapstructure:"secret" required:"true"`
}

type allKinds struct {
	Str     string            `mapstructure:"str" required:"true"`
	Int     int               `mapstructure:"int" required:"true"`
	Uint    uint              `mapstructure:"uint" required:"true"`
	Float   float64           `mapstructure:"float" required:"true"`
	Bool    bool              `mapstructure:"bool" required:"true"`
	Slice   []string          `mapstructure:"slice" required:"true"`
	Map     map[string]string `mapstructure:"map" required:"true"`
	Ptr     *int              `mapstructure:"ptr" required:"true"`
	Dur     int64             `json:"dur" validate:"required"`
	Skipped string            `mapstructure:"-" required:"true"`
	Plain   string            `required:"true"`
	Inner   nested            `mapstructure:"inner"`
	InnerP  *nested           `mapstructure:"innerp"`
	private string
}

func TestLoad_RequiredFieldKinds(t *testing.T) {
	_ = allKinds{}.private // silence unused: unexported fields must be ignored by validation

	var cfg allKinds
	err := Load("", &cfg)
	if errCode(t, err) != stderr.CodeValidationFailed {
		t.Fatalf("want CodeValidationFailed, got %v", err)
	}
	var se *stderr.Error
	errors.As(err, &se)
	got := map[string]bool{}
	for _, d := range se.Details {
		got[d.Field] = true
	}
	for _, want := range []string{"str", "int", "uint", "float", "bool", "slice", "map", "ptr", "dur", "plain", "inner.secret"} {
		if !got[want] {
			t.Errorf("missing required-field error for %q (got %v)", want, got)
		}
	}
	for _, unwanted := range []string{"-", "skipped", "private", "innerp.secret"} {
		if got[unwanted] {
			t.Errorf("unexpected error for %q", unwanted)
		}
	}
}

func TestLoad_RequiredSatisfied(t *testing.T) {
	file := writeFile(t, t.TempDir(), "ok.yaml", `
str: a
int: 1
uint: 2
float: 1.5
bool: true
slice: [x]
map: {k: v}
ptr: 3
dur: 4
plain: p
inner: {secret: s}
innerp: {secret: s}
`)
	var cfg allKinds
	if err := Load("", &cfg, WithConfigFile(file)); err != nil {
		t.Fatalf("all required fields are set, got %v", err)
	}
	if cfg.Inner.Secret != "s" || cfg.InnerP == nil || cfg.InnerP.Secret != "s" || *cfg.Ptr != 3 {
		t.Errorf("unexpected result %+v", cfg)
	}
}

func TestFieldTagKey(t *testing.T) {
	type s struct {
		A string `mapstructure:"a_map,omitempty"`
		B string `json:"b_json,omitempty"`
		C string
	}
	typ := reflect.TypeOf(s{})
	tests := []struct {
		idx  int
		want string
	}{{0, "a_map"}, {1, "b_json"}, {2, "c"}}
	for _, tt := range tests {
		if got := fieldTagKey(typ.Field(tt.idx)); got != tt.want {
			t.Errorf("field %d key = %q, want %q", tt.idx, got, tt.want)
		}
	}
}

func TestIsZeroValue(t *testing.T) {
	type pair struct{ A int }
	var nilIface any
	tests := []struct {
		name string
		v    any
		want bool
	}{
		{"blank string", "  ", true}, {"string", "x", false},
		{"int zero", 0, true}, {"int8", int8(1), false},
		{"uint zero", uint(0), true}, {"float", 0.5, false},
		{"bool false", false, true}, {"slice empty", []int{}, true}, {"map filled", map[string]int{"a": 1}, false},
		{"nil pointer", (*int)(nil), true}, {"struct zero", pair{}, true}, {"struct filled", pair{1}, false},
		{"nil interface", nilIface, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isZeroValue(reflect.ValueOf(tt.v)); got != tt.want {
				t.Errorf("isZeroValue(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}
