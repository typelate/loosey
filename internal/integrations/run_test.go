package integrations

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var coverDir string

func TestMain(m *testing.M) {
	var err error
	coverDir, err = os.MkdirTemp("", "loosey-cover-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating temp cover dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(coverDir)

	code := m.Run()

	// m.Run() has already written the parent's -test.coverprofile.
	// Append integration sub-module coverage data to it so tools
	// like GoLand see full coverage including out-of-process tests.
	if f := flag.Lookup("test.coverprofile"); f != nil && f.Value.String() != "" {
		if err := appendProfiles(coverDir, f.Value.String()); err != nil {
			fmt.Fprintf(os.Stderr, "appending integration coverage: %v\n", err)
		}
	}

	os.Exit(code)
}

func Test(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	modMatches, err := filepath.Glob("*/go.mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range modMatches {
		dir := filepath.Dir(match)
		t.Run(dir, func(t *testing.T) {
			t.Parallel()
			coverFile, _ := filepath.Abs(filepath.Join(coverDir, dir+".cover"))
			args := []string{
				"-C", dir, "test", "-race",
				"-coverpkg=github.com/typelate/loosey/...",
				"-coverprofile=" + coverFile,
			}
			if testing.Verbose() {
				args = append(args, "-v")
			}
			cmd := exec.CommandContext(t.Context(), "go", args...)
			out := t.Output()
			cmd.Stderr = out
			cmd.Stdout = out
			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// appendProfiles reads all .cover files from srcDir and appends coverage
// lines to dstFile (the parent's coverprofile, already written by m.Run).
func appendProfiles(srcDir, dstFile string) error {
	profiles, err := filepath.Glob(filepath.Join(srcDir, "*.cover"))
	if err != nil || len(profiles) == 0 {
		return err
	}

	out, err := os.OpenFile(dstFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	for _, p := range profiles {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || len(line) >= 5 && line[:5] == "mode:" {
				continue
			}
			if _, err := fmt.Fprintln(out, line); err != nil {
				_ = f.Close()
				return err
			}
		}
		_ = f.Close()
	}
	return nil
}
