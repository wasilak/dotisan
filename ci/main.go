package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"dagger.io/dagger"
)

const (
	goImage    = "golang:1.26.2"
	binaryName = "dotisan"
)

type target struct{ os, arch string }

var targets = []target{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
}

func main() {
	var (
		doCI    = flag.Bool("ci", false, "run vet + test + build (shorthand for all three)")
		doVet   = flag.Bool("vet", false, "run go vet ./...")
		doTest  = flag.Bool("test", false, "run unit tests (go test -race ./...)")
		doBuild = flag.Bool("build", false, "cross-compile binaries for all targets")
		version = flag.String("version", "dev", "version string injected via -ldflags")
		output  = flag.String("output", "../dist", "output directory for binaries (relative to ci/)")
	)
	flag.Parse()

	if *doCI {
		*doVet = true
		*doTest = true
		*doBuild = true
	}

	if err := run(context.Background(), *doVet, *doTest, *doBuild, *version, *output); err != nil {
		fmt.Fprintln(os.Stderr, "pipeline failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, doVet, doTest, doBuild bool, version, outputDir string) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return fmt.Errorf("dagger connect: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory("..")

	base := client.Container().
		From(goImage).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", client.CacheVolume("dotisan-go-mod")).
		WithMountedCache("/root/.cache/go-build", client.CacheVolume("dotisan-go-build"))

	if doVet {
		if err := vet(ctx, base); err != nil {
			return err
		}
	}
	if doTest {
		if err := test(ctx, base); err != nil {
			return err
		}
	}
	if doBuild {
		if err := build(ctx, base, version, outputDir); err != nil {
			return err
		}
	}
	return nil
}

func vet(ctx context.Context, base *dagger.Container) error {
	fmt.Println("→ go vet ./...")
	_, err := base.WithExec([]string{"go", "vet", "./..."}).Sync(ctx)
	if err != nil {
		return fmt.Errorf("go vet: %w", err)
	}
	fmt.Println("✓ vet passed")
	return nil
}

func test(ctx context.Context, base *dagger.Container) error {
	fmt.Println("→ go test -race ./...")
	_, err := base.WithExec([]string{"go", "test", "-race", "-v", "./..."}).Sync(ctx)
	if err != nil {
		return fmt.Errorf("go test: %w", err)
	}
	fmt.Println("✓ tests passed")
	return nil
}

func build(ctx context.Context, base *dagger.Container, version, outputDir string) error {
	fmt.Printf("→ building %d targets (version=%s)\n", len(targets), version)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outputDir, err)
	}

	ldflags := fmt.Sprintf("-X github.com/wasilak/dotisan/cmd.Version=%s -s -w", version)

	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(targets))
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(t target) {
			defer wg.Done()
			name := fmt.Sprintf("%s-%s-%s", binaryName, t.os, t.arch)
			outPath := "/tmp/" + name
			_, err := base.
				WithEnvVariable("GOOS", t.os).
				WithEnvVariable("GOARCH", t.arch).
				WithEnvVariable("CGO_ENABLED", "0").
				WithExec([]string{"go", "build", "-ldflags", ldflags, "-o", outPath, "."}).
				File(outPath).
				Export(ctx, filepath.Join(outputDir, name))
			results <- result{name, err}
		}(t)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var errs []string
	for r := range results {
		if r.err != nil {
			errs = append(errs, r.err.Error())
		} else {
			fmt.Println(" ✓", r.name)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("build failures:\n%s", strings.Join(errs, "\n"))
	}
	fmt.Printf("✓ binaries written to %s/\n", outputDir)
	return nil
}
