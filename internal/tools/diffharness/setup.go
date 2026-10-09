package diffharness

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultOracleURL = "http://127.0.0.1:8081"
	defaultRefDir    = "/opt/gopress-ref"
	dbUser           = "gopress"
	dbPass           = "gopress"
)

// SetupOptions controls DB/uploads/oracle preparation.
type SetupOptions struct {
	RepoRoot string
	RefDir   string
	SkipDB   bool
}

// PrepareFixture loads site.sql into both DBs, copies uploads, writes wp-config.php,
// and ensures the PHP oracle is reachable.
func PrepareFixture(opts SetupOptions) error {
	root := opts.RepoRoot
	refDir := opts.RefDir
	if refDir == "" {
		refDir = defaultRefDir
	}
	sqlPath := filepath.Join(root, "tests/fixtures/site.sql")
	uploads := filepath.Join(root, "tests/fixtures/uploads")

	if !opts.SkipDB {
		if err := importDB("gopress_ref", sqlPath); err != nil {
			return fmt.Errorf("gopress_ref: %w", err)
		}
		if err := importDB("gopress_go", sqlPath); err != nil {
			return fmt.Errorf("gopress_go: %w", err)
		}
	}

	if err := copyUploads(uploads, filepath.Join(refDir, "wp-content/uploads")); err != nil {
		return fmt.Errorf("ref uploads: %w", err)
	}
	if err := copyUploads(uploads, filepath.Join(root, "wp-content/uploads")); err != nil {
		return fmt.Errorf("go uploads: %w", err)
	}

	if err := writeGoWPConfig(root); err != nil {
		return err
	}

	if err := ensureOracle(root); err != nil {
		return err
	}
	return nil
}

func importDB(name, sqlPath string) error {
	if _, err := os.Stat(sqlPath); err != nil {
		return err
	}
	// Recreate schema for a clean import.
	drop := exec.Command("mysql", "-u"+dbUser, "-p"+dbPass, "-e",
		fmt.Sprintf("DROP DATABASE IF EXISTS %s; CREATE DATABASE %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", name, name))
	if out, err := drop.CombinedOutput(); err != nil {
		return fmt.Errorf("recreate %s: %w\n%s", name, err, out)
	}
	f, err := os.Open(sqlPath)
	if err != nil {
		return err
	}
	defer f.Close()
	cmd := exec.Command("mysql", "-u"+dbUser, "-p"+dbPass, "--default-character-set=utf8mb4", name)
	cmd.Stdin = f
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("import %s: %w\n%s", name, err, out)
	}
	return nil
}

func copyUploads(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func writeGoWPConfig(root string) error {
	tmplPath := filepath.Join(root, "tools/oracle/wp-config.php.tmpl")
	raw, err := os.ReadFile(tmplPath)
	if err != nil {
		return err
	}
	out := strings.Replace(string(raw), "define( 'DB_NAME', 'gopress_ref' );", "define( 'DB_NAME', 'gopress_go' );", 1)
	// Point ABSPATH at the workspace checkout when this file lives at repo root.
	dest := filepath.Join(root, "wp-config.php")
	return os.WriteFile(dest, []byte(out), 0o644)
}

func ensureOracle(root string) error {
	if err := waitURL(defaultOracleURL+"/wp-login.php", 2*time.Second); err == nil {
		return nil
	}
	cmd := exec.Command("bash", filepath.Join(root, "scripts/oracle-up.sh"))
	cmd.Dir = root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("oracle-up: %w", err)
	}
	return waitURL(defaultOracleURL+"/wp-login.php", 30*time.Second)
}

func waitURL(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		cmd := exec.Command("curl", "-sf", "-o", "/dev/null", url)
		if err := cmd.Run(); err == nil {
			return nil
		} else {
			last = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if last == nil {
		last = fmt.Errorf("timeout")
	}
	return fmt.Errorf("wait %s: %w", url, last)
}
