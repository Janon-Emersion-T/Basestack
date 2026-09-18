package migrations

import (
	"os"
	"path/filepath"
	"testing"
)

func sqlFile(t *testing.T, dir, name, sql string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(sql), 0644); err != nil {
		t.Fatal(err)
	}
}
func TestDiscoveryAndCreation(t *testing.T) {
	dir := t.TempDir()
	sqlFile(t, dir, "000002_second.sql", "SELECT 2;")
	sqlFile(t, dir, "000001_first.sql", "SELECT 1;")
	files, err := Discover(dir)
	if err != nil || len(files) != 2 || files[0].Version != 1 || len(files[0].Checksum) != 64 {
		t.Fatal("discovery/order failed", err)
	}
	path, err := Create(dir, "create_example")
	if err != nil || filepath.Base(path) != "000003_create_example.sql" {
		t.Fatal("creation failed", err)
	}
	if _, err := Create(dir, "create_example"); err == nil {
		t.Fatal("duplicate name accepted")
	}
	if _, err := Create(dir, "../unsafe"); err == nil {
		t.Fatal("unsafe name accepted")
	}
	sqlFile(t, dir, "000003_conflict.sql", "SELECT 1;")
	if _, err := Discover(dir); err == nil {
		t.Fatal("duplicate version accepted")
	}
}
func TestBadFilesAndSQL(t *testing.T) {
	for _, name := range []string{"initial.sql", "000000_bad.sql", "000001_BAD.sql"} {
		dir := t.TempDir()
		sqlFile(t, dir, name, "SELECT 1;")
		if _, err := Discover(dir); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "source.sql")
	os.WriteFile(target, []byte("SELECT 1;"), 0644)
	os.Symlink(target, filepath.Join(dir, "000001_link.sql"))
	if _, err := Discover(dir); err == nil {
		t.Fatal("symlink accepted")
	}
	for _, sql := range []string{"BEGIN; SELECT 1; COMMIT;", "/*comment*/COMMIT;", "SELECT 1; END;", "PREPARE TRANSACTION 'x';", "START TRANSACTION;", "SELECT 'unterminated", "/* broken", "DO $$unterminated"} {
		if transactionFree(sql) == nil {
			t.Fatalf("accepted SQL %q", sql)
		}
	}
	for _, sql := range []string{"-- no tables yet", "SELECT 'COMMIT;';", "DO $$ BEGIN PERFORM 1; END $$;", `SELECT "commit" FROM example;`, "/* a /* nested */ comment */ SELECT 1;", `SELECT E'escaped\' COMMIT';`} {
		if err := transactionFree(sql); err != nil {
			t.Fatalf("rejected SQL %q: %v", sql, err)
		}
	}
}
func TestHistoryIntegrity(t *testing.T) {
	files := []File{{Version: 1, Name: "first", Checksum: "abc"}, {Version: 2, Name: "second", Checksum: "def"}}
	if err := Integrity(files, files[:1]); err != nil {
		t.Fatal(err)
	}
	for _, history := range [][]File{{{Version: 1, Name: "first", Checksum: "changed"}}, {{Version: 1, Name: "renamed", Checksum: "abc"}}, {files[1]}, append(append([]File{}, files...), File{Version: 3})} {
		if Integrity(files, history) == nil {
			t.Fatal("corrupt history accepted")
		}
	}
}
