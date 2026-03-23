package db

import (
	"database/sql"
	"embed"
	"errors"
	"io"
	"os"
	"path/filepath"
)

//go:embed views.sql
//go:embed fasoladb/minutes.db
//go:embed fasoladb/minutes_schema.sql
var packaged embed.FS

func getDBpath() string { return filepath.Join(os.TempDir(), "minutes.db") }

func Init() error {
	dbPath := getDBpath()
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		return nil
	}

	f, err := packaged.Open("fasoladb/minutes.db")
	if err != nil {
		return err
	}
	defer f.Close()

	target, err := os.Create(dbPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(target, f); err != nil {
		return errors.Join(err, target.Close())
	}
	if err := target.Close(); err != nil {
		return err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	setupSql, err := packaged.ReadFile("views.sql")
	if err != nil {
		return err
	}

	if _, err := db.Exec(string(setupSql)); err != nil {
		return err
	}

	return nil
}
