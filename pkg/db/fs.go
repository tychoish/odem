package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem"
	"github.com/tychoish/odem/pkg/release"
)

//go:embed views.sql
//go:embed setup.sql
//go:embed fasoladb/minutes.db
//go:embed fasoladb/minutes_schema.sql
var packaged embed.FS

func Reset() error {
	dbPath := getDBpath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(dbPath)
}

func Init(ctx context.Context) (err error) {
	var ec erc.Collector
	defer func() { ec.Push(err); err = ec.Resolve() }()
	dbPath := getDBpath()
	// if the database exists, in /tmp we can just use it as a timesaving measure.
	conf := odem.GetConfiguration(ctx)
	grip.Warning(grip.When(conf == nil, "configuration is nil!"))
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		odemMtime := odemBinaryMtime()
		dbMtime := odemTempDBMtime()
		// compare the mtimes; if the binry is compiled and newer than the database; we
		// should rebuild the database. otherwise the database was built with this binary
		// and we can just run the idempotent index creation (for the 'go run' case) and use
		// the database otherwise we fall through and set up the local database
		if (conf != nil && !conf.Settings.ManualReloadDB) || os.Getenv("ODEM_DEVLOP") != "" && (dbMtime.After(odemMtime) || odemMtime.IsZero()) {
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				return err
			}
			defer ec.Check(db.Close)

			setupSql, err := packaged.ReadFile("views.sql")
			if err != nil {
				return err
			}

			if _, err := db.Exec(string(setupSql)); err != nil {
				return err
			}

			return nil
		}
	}

	grip.Info(grip.MPrintln("setting up the local minutes database:", dbPath))
	f, err := packaged.Open("fasoladb/minutes.db")
	if err != nil {
		return err
	}
	defer ec.Check(f.Close)

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

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer ec.Check(db.Close)

	grip.Info("applying odem specific modifications")
	for file := range irt.Args("setup.sql", "views.sql") {
		grip.Info(grip.MPrintln("reading", file))
		setupSql, err := packaged.ReadFile(file)
		if ec.PushOk(err) {
			grip.Info(grip.MPrintln("applying", file))
			if _, err := db.Exec(string(setupSql)); err != nil {
				ec.Push(err)
			}
		}
	}

	grip.Info("database initialized")
	return nil
}

func odemBinaryMtime() time.Time {
	if ex, err := os.Executable(); err == nil && strings.Contains(ex, "go-build") {
		return time.Time{}
	} else if stat, err := os.Stat(ex); err == nil {
		return stat.ModTime()
	}

	if len(os.Args) == 0 || os.Args[0] == "go" || !strings.HasPrefix(os.Args[0], release.Name) {
		return time.Time{}
	}

	for p := range irt.RemoveZeros(irt.Args(
		lookInPath(release.Name),
		lookInPath(os.Args[0]),
		filepath.Join(pwd(), release.Name),
		filepath.Join(pwd(), os.Args[0]),
	)) {
		stat, err := os.Stat(p)
		if os.IsNotExist(err) || stat == nil {
			continue
		}
		grip.Notice(grip.MPrintln("fallback", p))
		return stat.ModTime()
	}

	// well we tried
	return time.Time{}
}

func odemTempDBMtime() time.Time {
	stat, err := os.Stat(getDBpath())
	if os.IsNotExist(err) || stat == nil {
		return time.Time{}
	}
	return stat.ModTime()
}

func getDBpath() string             { return dbpath }
func pwd() string                   { v, _ := os.Getwd(); return v }
func lookInPath(name string) string { v, _ := exec.LookPath(name); return v }
