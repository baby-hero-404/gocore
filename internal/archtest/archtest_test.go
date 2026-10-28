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
	Name       string
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
	"gorm.io/gorm":                        {"database", "database/databasemock", "database/pagination", "database/postgres", "database/mysql", "database/sqlite"},
	"gorm.io/driver/postgres":             {"database/postgres"},
	"gorm.io/driver/mysql":                {"database/mysql"},
	"gorm.io/driver/sqlite":               {"database/sqlite"},
	"google.golang.org/grpc":              {"grpcserver"}, // reserved: gRPC is not implemented yet
	"connectrpc.com":                      {"grpcserver"}, // reserved: gRPC is not implemented yet
	"github.com/spf13/viper":              {"env"},
	"github.com/go-playground/validator":  {"stdres"},
	"github.com/golang-jwt/jwt":           {"iam"},
	"github.com/stretchr/testify":         {}, // only *mock packages, checked separately
}

// rank orders top-level packages bottom-up: a package may only import gocore packages of
// equal or lower rank, or packages under its own top-level directory.
var rank = map[string]int{
	"stderr": 0, "stdlog": 1, "stdres": 2, "env": 2, "monitor": 2, "httpmw": 3,
	"database": 3, "cache": 3, "queue": 3, "lock": 4, "httpserver": 4, "grpcserver": 4, "iam": 5,
}

// infra lists the infrastructure tiers. They are independent of each other: a package in this
// set may import another one only through an explicit entry in infraAllowed (importer -> imported).
var infra = map[string]bool{"database": true, "cache": true, "queue": true, "lock": true, "httpserver": true, "grpcserver": true}

var infraAllowed = map[string][]string{"lock": {"cache"}}

// iamForbidden are packages iam must never import; it depends on consumer interfaces instead.
var iamForbidden = []string{"database", "cache", "queue", "lock"}

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

// parseNonTest parses the non-test Go files of p, skipping generated mock_*.go files.
func parseNonTest(t *testing.T, p pkg) map[string]*ast.File {
	t.Helper()
	files := map[string]*ast.File{}
	for _, f := range p.GoFiles {
		if isMockPkg(rel(p.ImportPath)) && strings.HasPrefix(f, "mock_") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(p.Dir, f), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		files[f] = file
	}
	return files
}

// TestNoPanicOrExit enforces CS-3 and CS-6: library code returns errors and never panics or
// terminates the process.
func TestNoPanicOrExit(t *testing.T) {
	for _, p := range listPackages(t) {
		if p.Name == "main" {
			continue // CS-6: only a main package may terminate the process (allowed solely in internal/tools)
		}
		for name, file := range parseNonTest(t, p) {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fn := call.Fun.(type) {
				case *ast.Ident:
					if fn.Name == "panic" {
						t.Errorf("%s/%s calls panic(): return an error instead (CS-3)", rel(p.ImportPath), name)
					}
				case *ast.SelectorExpr:
					id, ok := fn.X.(*ast.Ident)
					if !ok {
						return true
					}
					if (id.Name == "log" && strings.HasPrefix(fn.Sel.Name, "Fatal")) || (id.Name == "os" && fn.Sel.Name == "Exit") {
						t.Errorf("%s/%s calls %s.%s: libraries must not terminate the process (CS-6)", rel(p.ImportPath), name, id.Name, fn.Sel.Name)
					}
				}
				return true
			})
		}
	}
}

// TestNoMainInLibrary enforces EXT-6: package main (demos, CLIs) lives only under internal/tools.
func TestNoMainInLibrary(t *testing.T) {
	for _, p := range listPackages(t) {
		r := rel(p.ImportPath)
		if p.Name == "main" && !strings.HasPrefix(r, "internal/tools") {
			t.Errorf("%s is package main: demos belong in Example* tests, tools in internal/tools (EXT-6)", r)
		}
	}
}

// TestMocksAssertInterface enforces EXT-4: every <pkg>mock package declares at least one
// compile-time assertion `var _ Iface = (*Mock)(nil)` in a hand-written file.
func TestMocksAssertInterface(t *testing.T) {
	for _, p := range listPackages(t) {
		if !isMockPkg(rel(p.ImportPath)) {
			continue
		}
		found := false
		for _, file := range parseNonTest(t, p) {
			for _, d := range file.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					vs := spec.(*ast.ValueSpec)
					if len(vs.Names) == 1 && vs.Names[0].Name == "_" && vs.Type != nil && len(vs.Values) == 1 {
						found = true
					}
				}
			}
		}
		if !found {
			t.Errorf("%s has no compile-time assertion `var _ Iface = (*Mock)(nil)` (EXT-4); add it to doc.go", rel(p.ImportPath))
		}
	}
}

// TestInfraPackagesStayIndependent enforces ARCH-5: infrastructure packages do not import each
// other except through infraAllowed, and iam never imports infrastructure packages directly.
func TestInfraPackagesStayIndependent(t *testing.T) {
	for _, p := range listPackages(t) {
		r := rel(p.ImportPath)
		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp, modulePath+"/") {
				continue
			}
			from, to := top(r), top(rel(imp))
			if from == to {
				continue
			}
			if infra[from] && infra[to] && !contains(infraAllowed[from], to) {
				t.Errorf("%s imports %s: infrastructure packages must stay independent (ARCH-5); add to infraAllowed only with an ADR", r, rel(imp))
			}
			if from == "iam" && contains(iamForbidden, to) {
				t.Errorf("%s imports %s: iam must depend on consumer interfaces, not infrastructure (ARCH-5)", r, rel(imp))
			}
		}
	}
}
