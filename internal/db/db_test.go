package db_test

import (
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/db"
)

func TestPrepare(t *testing.T) {
	q, args, err := db.Prepare(
		"SELECT * FROM wp_posts WHERE ID = %d AND post_status = %s AND score > %f",
		42, "publish", 1.5,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT * FROM wp_posts WHERE ID = ? AND post_status = ? AND score > ?"
	if q != want {
		t.Fatalf("query=%q want %q", q, want)
	}
	if len(args) != 3 || args[0] != 42 || args[1] != "publish" || args[2] != 1.5 {
		t.Fatalf("args=%v", args)
	}
}

func TestPrepareQuotesAndPercent(t *testing.T) {
	q, args, err := db.Prepare("SELECT '%s' AS a, %% AS pct", "x")
	if err != nil {
		t.Fatal(err)
	}
	// Leading quotes around %s are stripped like wpdb; %% → literal %.
	if q != "SELECT ? AS a, % AS pct" {
		t.Fatalf("query=%q", q)
	}
	if len(args) != 1 || args[0] != "x" {
		t.Fatalf("args=%v", args)
	}
}

func TestPrepareArgMismatch(t *testing.T) {
	if _, _, err := db.Prepare("SELECT %s", 1, 2); err == nil {
		t.Fatal("expected unused-arg error")
	}
	if _, _, err := db.Prepare("SELECT %s AND %d", "a"); err == nil {
		t.Fatal("expected not-enough-args error")
	}
	if _, _, err := db.Prepare("SELECT %x", 1); err == nil {
		t.Fatal("expected unsupported placeholder")
	}
}

func TestEscapeLike(t *testing.T) {
	got := db.EscapeLike(`50%_off\sale`)
	want := `50\%\_off\\sale`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestTableHelpers(t *testing.T) {
	d := &db.DB{TablePrefix: "wp_"}
	if d.Posts() != "wp_posts" || d.Options() != "wp_options" {
		t.Fatalf("posts=%s options=%s", d.Posts(), d.Options())
	}
	d.TablePrefix = "gp_"
	if d.Users() != "gp_users" || d.PostMeta() != "gp_postmeta" {
		t.Fatalf("users=%s postmeta=%s", d.Users(), d.PostMeta())
	}
	if d.TermRelationships() != "gp_term_relationships" {
		t.Fatalf("term_relationships=%s", d.TermRelationships())
	}
}

func TestBlognameIntegration(t *testing.T) {
	var (
		handle *db.DB
		err    error
	)
	if dsn := os.Getenv("GOPRESS_TEST_DSN"); dsn != "" {
		handle, err = db.OpenDSN(dsn, "wp_", "utf8mb4")
	} else {
		cfg := &config.Config{
			DBName:      "gopress_ref",
			DBUser:      "gopress",
			DBPassword:  "gopress",
			DBHost:      "127.0.0.1:3306",
			DBCharset:   "utf8mb4",
			TablePrefix: "wp_",
		}
		handle, err = db.Open(cfg)
	}
	if err != nil {
		t.Skipf("DB unavailable: %v", err)
	}
	defer handle.Close()

	q, args, err := handle.Prepare(
		"SELECT option_value FROM "+handle.Options()+" WHERE option_name = %s",
		"blogname",
	)
	if err != nil {
		t.Fatal(err)
	}
	v, err := handle.GetVar(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := v.(string)
	if s == "" {
		// driver may return []byte; GetVar converts via Scan into any
		if b, ok := v.([]byte); ok {
			s = string(b)
		}
	}
	if s == "" {
		t.Fatalf("blogname empty; got %#v", v)
	}
	t.Logf("blogname=%q", s)
}
