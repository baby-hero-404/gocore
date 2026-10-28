// Command covercheck enforces the coverage ratchet (TEST-3): a package's statement coverage may
// not drop below its recorded baseline, and packages without a baseline must reach 80%.
//
// Usage:
//
//	go test ./... -race -covermode=atomic -coverprofile=coverage.out
//	go run ./internal/tools/covercheck -profile coverage.out -baseline coverage-baseline.txt
//	go run ./internal/tools/covercheck -profile coverage.out -baseline coverage-baseline.txt -update
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	profilePath := flag.String("profile", "coverage.out", "coverage profile produced by go test -coverprofile")
	baselinePath := flag.String("baseline", "coverage-baseline.txt", "per-package baseline file")
	update := flag.Bool("update", false, "rewrite the baseline from the current profile")
	flag.Parse()

	if err := run(*profilePath, *baselinePath, *update); err != nil {
		fmt.Fprintln(os.Stderr, "covercheck:", err)
		os.Exit(1)
	}
}

func run(profilePath, baselinePath string, update bool) error {
	profile, err := os.Open(profilePath) //nolint:gosec // path comes from the developer's command line
	if err != nil {
		return fmt.Errorf("open profile: %w", err)
	}
	defer func() { _ = profile.Close() }()

	current, err := parseProfile(profile)
	if err != nil {
		return err
	}
	all, err := listPackages()
	if err != nil {
		return err
	}
	for _, pkg := range all {
		if _, ok := current[pkg.ImportPath]; !ok && measured(pkg.ImportPath) && hasStatements(pkg.Dir, pkg.GoFiles) {
			current[pkg.ImportPath] = 0 // has code but no tests at all
		}
	}

	if update {
		return writeBaseline(baselinePath, current)
	}

	baseline, err := readBaseline(baselinePath)
	if err != nil {
		return err
	}
	problems := evaluate(current, baseline)
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "  -", p)
	}
	if len(problems) > 0 {
		return fmt.Errorf("%d coverage problem(s); raise coverage or, for an intentional change, run with -update", len(problems))
	}
	fmt.Printf("covercheck: %d packages at or above baseline\n", len(current))
	return nil
}

type listedPackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
}

// listPackages returns every package in the module, so packages without tests count as 0%.
func listPackages() ([]listedPackage, error) {
	out, err := exec.CommandContext(context.Background(), "go", "list", "-json", "./...").Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}
	var pkgs []listedPackage
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}
