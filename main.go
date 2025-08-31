package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/alecthomas/kong"
	"github.com/jmoiron/sqlx"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/commands/service"
	"github.com/rcy/whatever/version"
)

func appDataFile(appName string, filename string) (string, error) {
	if !version.IsRelease() {
		appName += "-devel"
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, appName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, filename), nil
}

func main() {
	dbFile, err := appDataFile("whatever-cli", "data.sqlite")
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Tune connection pool
	db.SetMaxOpenConns(1) // SQLite supports one writer, so cap to 1
	db.SetMaxIdleConns(1)

	// Enable WAL mode for better concurrency and durability
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		log.Fatal("failed to enable WAL mode:", err)
	}

	// Optional performance pragmas (tweak based on needs):
	if _, err := db.Exec(`PRAGMA synchronous = NORMAL;`); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		log.Fatal(err)
	}

	// Example schema
	if _, err := db.Exec(`
		create table if not exists events (
			event_id integer primary key autoincrement,
                        created_at timestamp not null default current_timestamp,
                        aggregate_type text not null,
                        aggregate_id text not null,
                        event_type text not null,
                        event_data text not null
		);
	`); err != nil {
		log.Fatal(err)
	}

	sqlxDB := sqlx.NewDb(db, "sqlite3")

	kctx := kong.Parse(&commands.CLI)
	err = kctx.Run(&service.Service{DB: sqlxDB, DBFile: dbFile})
	kctx.FatalIfErrorf(err)
}
