package databasev2

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	migrate "github.com/uptrace/bun/migrate"

	"github.com/tozny/utils-go/logging"
)

var (
	// ErrorNoRows should be returned whenever no rows were found for the given query
	ErrorNoRows = sql.ErrNoRows
)

// DB wraps a Bun client and mirrors the behaviour of the original DB type
// implemented with go‑pg, but uses Bun under the hood.
//
// The public API stays minimal and version-specific via package isolation.
type DB struct {
	Client      *bun.DB
	Logger      logging.Logger
	initializer func(*DB)
}

// DBConfig wraps config for connecting to a database.
type DBConfig struct {
	Address       string
	User          string
	Database      string
	Password      string
	Logger        logging.Logger
	EnableLogging bool
	EnableTLS     bool
	SkipVerifyTLS bool
}

// New returns a Bun‑backed DB using the supplied configuration.
func New(cfg DBConfig) DB {

	u := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   cfg.Address,
		Path:   cfg.Database,
	}

	q := u.Query()
	if !cfg.EnableTLS {
		q.Set("sslmode", "disable")
	}
	u.RawQuery = q.Encode()

	drvOpts := []pgdriver.Option{pgdriver.WithDSN(u.String())}
	if cfg.EnableTLS {
		drvOpts = append(drvOpts, pgdriver.WithTLSConfig(&tls.Config{InsecureSkipVerify: cfg.SkipVerifyTLS}))
	}

	sqlDB := sql.OpenDB(pgdriver.NewConnector(drvOpts...))
	bunDB := bun.NewDB(sqlDB, pgdialect.New())

	if cfg.EnableLogging {
		bunDB.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	return DB{
		Client:      bunDB,
		Logger:      cfg.Logger,
		initializer: nil,
	}
}

// Close shuts down connections held by the underlying *bun.DB.
func (db *DB) Close() {
	db.Logger.Debug("DB: closing database connection (Bun/v2)")
	err := db.Client.Close()
	if err != nil {
		db.Logger.Errorf("DB: error while closing db connection: %+v", err)
	}
}

// Migrate applies migrations passed by the calling service.
func (db *DB) Migrate(migrations *migrate.Migrations) error {
	ctx := context.Background()
	migrator := migrate.NewMigrator(db.Client, migrations, migrate.WithMarkAppliedOnSuccess(true))

	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("DB: migration init failed: %w", err)
	}

	group, err := migrator.Migrate(ctx)
	if err != nil {
		// If at least one migration was attempted, the last one in the group is likely the failure
		if len(group.Migrations) > 0 {
			last := group.Migrations[len(group.Migrations)-1]
			return fmt.Errorf("DB: migration %q failed: %w", last.Name, err)
		}
		// Else generic panic
		return fmt.Errorf("DB: migration failed before applying any: %w", err)
	}

	if len(group.Migrations) == 0 {
		fmt.Println("DB: No new migrations to apply.")
		return nil
	}

	for _, m := range group.Migrations {
		fmt.Printf("DB: Applied migration: %s\n", m.Name)
	}
	return err
}

// RunMigrations retries migrations using a supplied migration collection.
func RunMigrations(db *DB, migrations *migrate.Migrations) {
	for {
		db.Logger.Debug("DB.RunMigrations: Running migrations.")
		if err := db.Migrate(migrations); err != nil {
			db.Logger.Errorf("DB.RunMigrations: %v, will retry in 1 second.", err)
			time.Sleep(1 * time.Second)
			continue
		}
		db.Logger.Debug("DB.RunMigrations: Migrations finished.")
		break
	}
}

// Initialize invokes the configured initializer (defaults to RunMigrations).
func (db *DB) Initialize() {
	if db.initializer != nil {
		db.initializer(db)
	}
}

// Initializer lets callers replace the default RunMigrations behaviour.
func (db *DB) Initializer(init func(*DB)) {
	db.initializer = init
}

// Ping verifies that the database is reachable.
func (db *DB) Ping() error {
	return db.Client.DB.Ping()
}
