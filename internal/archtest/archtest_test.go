package archtest

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/baby-hero-404/gocore"

type pkg struct {
	ImportPath string
	Dir        string
	Doc        string
	GoFiles    []string
	Imports    []string
}

// vendorOwners lists, per third-party import prefix, the only packages allowed to import it.
// Anything else must depend on the owner's interface instead.
var vendorOwners = map[string][]string{
	"github.com/gin-gonic/gin":            {"httpserver", "httpserver/httpservermock", "stdres/stdgin", "iam/iamgin", "monitor/promgin"},
	"firebase.google.com/go":              {"iam/firebase"},
	"github.com/getsentry/sentry-go":      {"monitor/sentry"},
	"github.com/Graylog2/go-gelf":         {"monitor/graylog"},
	"github.com/prometheus/client_golang": {"monitor/prom"},
	"github.com/segmentio/kafka-go":       {"queue/kafka/consumer", "queue/kafka/producer"},
	"github.com/redis/go-redis":           {"cache", "cache/cachemock", "lock"},
	"github.com/go-redsync/redsync":       {"lock"},
	"gorm.io":                             {"database", "database/databasemock"},
	"github.com/spf13/viper":              {"env"},
	"github.com/go-playground/validator":  {"stdres"},
	"github.com/golang-jwt/jwt":           {"iam"},
	"github.com/stretchr/testify":         {}, // only *mock packages, checked separately
}

// rank orders top-level packages bottom-up: a package may only import gocore packages of
// equal or lower rank, or packages under its own top-level directory.
var rank = map[string]int{
	"stderr": 0, "stdlog": 1, "stdres": 2, "env": 2, "monitor": 2,
	"database": 3, "cache": 3, "queue": 3, "lock": 4, "httpserver": 4, "iam": 5,
}

func listPackages(t *testing.T) []pkg {
	t.Helper()
	out, err := exec.Command("go", "list", "-json", modulePath+"/...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	var pkgs []pkg
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

func rel(importPath string) string {
	return strings.TrimPrefix(strings.TrimPrefix(importPath, modulePath), "/")
}

func top(rel string) string {
	return strings.SplitN(rel, "/", 2)[0]
}

func isMockPkg(rel string) bool {
	return strings.HasSuffix(rel, "mock")
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestVendorImportsStayInTheirOwners(t *testing.T) {
	for _, p := range listPackages(t) {
		r := rel(p.ImportPath)
		for _, imp := range p.Imports {
			for prefix, owners := range vendorOwners {
				if !strings.HasPrefix(imp, prefix) {
					continue
				}
				if prefix == "github.com/stretchr/testify" {
					if !isMockPkg(r) {
						t.Errorf("%s imports %s: testify is only allowed in <pkg>mock packages", r, imp)
					}
					continue
				}
				if !contains(owners, r) {
					t.Errorf("%s imports %s: only %v may import it; put adapters in a sub-package", r, imp, owners)
				}
			}
		}
	}
}

func TestDependencyDirection(t *testing.T) {
	for _, p := range listPackages(t) {
		r := rel(p.ImportPath)
		own, ok := rank[top(r)]
		if !ok {
			if r != "" && !strings.HasPrefix(r, "internal") {
				t.Errorf("%s: top-level package %q has no rank in archtest; add it", r, top(r))
			}
			continue
		}
		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp, modulePath+"/") {
				continue
			}
			ir := rel(imp)
			if top(ir) == top(r) {
				continue
			}
			if rank[top(ir)] > own {
				t.Errorf("%s (rank %d) imports %s (rank %d): dependencies must point down", r, own, ir, rank[top(ir)])
			}
		}
	}
}

func TestEveryPackageHasDoc(t *testing.T) {
	for _, p := range listPackages(t) {
		if strings.TrimSpace(p.Doc) == "" {
			t.Errorf("%s has no package doc comment (add doc.go)", rel(p.ImportPath))
		}
	}
}

func TestNoInitFunctions(t *testing.T) {
	for _, p := range listPackages(t) {
		for _, f := range p.GoFiles {
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(p.Dir, f), nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", f, err)
			}
			for _, d := range file.Decls {
				if fn, ok := declFunc(d); ok && fn == "init" {
					t.Errorf("%s/%s declares init(): gocore forbids init side effects", rel(p.ImportPath), f)
				}
			}
		}
	}
}

func declFunc(d ast.Decl) (string, bool) {
	fn, ok := d.(*ast.FuncDecl)
	if !ok || fn.Recv != nil {
		return "", false
	}
	return fn.Name.Name, true
}
