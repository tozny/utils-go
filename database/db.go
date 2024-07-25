package database

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/go-pg/pg/v10"
	migrations "github.com/robinjoseph08/go-pg-migrations/v3"
	"github.com/tozny/utils-go/logging"
)

var (
	// ErrorNoRows should be returned whenever no rows were found for the given query
	ErrorNoRows = pg.ErrNoRows
)

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

// DB wraps a client for a database.
type DB struct {
	Client      *pg.DB
	Logger      logging.Logger
	initializer func(*DB)
}

// Close closes a connection to a database. Once close has been called calling other methods on db will error.
func (db *DB) Close() {
	db.Logger.Debug("Closing database connection")
	db.Client.Close()
}

// Migrate runs all migrations found in migrationDir against db.
func (db *DB) Migrate() error {
	err := migrations.Run(db.Client, "", []string{"", "migrate"})
	return err
}

// dbLogger implements the DBLogger interface for the go-pg module
type dbLogger struct {
	logger logging.Logger
}

// context key for query timing context
var dlTimingKey struct{} = struct{}{}

// BeforeQuery is called before a query is executed.
func (d dbLogger) BeforeQuery(ctx context.Context, q *pg.QueryEvent) (context.Context, error) {
	return context.WithValue(ctx, dlTimingKey, time.Now()), nil
}

// AfterQuery is called after a query is executed.
func (d dbLogger) AfterQuery(ctx context.Context, q *pg.QueryEvent) error {
	query, err := q.FormattedQuery()
	if err != nil {
		d.logger.Errorf("error %q formatting query:\n%+v ", err, query)
		return nil
	}
	start, ok := ctx.Value(dlTimingKey).(time.Time)
	if !ok {
		d.logger.Errorf("Unable find timing context in query:\n%+v ", query)
		return nil
	}
	d.logger.Infof("executed query in %s:\n%+v", time.Since(start), string(query))
	return nil
}

// New returns a new DB object which wraps a connection to the database specified in config
func New(config DBConfig) DB {
	options := &pg.Options{
		Addr:     config.Address,
		User:     config.User,
		Database: config.Database,
		Password: config.Password,
	}
	if config.EnableTLS {
		options.TLSConfig = &tls.Config{InsecureSkipVerify: config.SkipVerifyTLS}
	}

	db := pg.Connect(options)
	if config.EnableLogging {
		db.AddQueryHook(dbLogger{logger: config.Logger})
	}
	return DB{
		Client:      db,
		Logger:      config.Logger,
		initializer: RunMigrations,
	}
}

// Initialize runs any needed set up operations for the database. This defaults
// to RunMigrations, but can be set using the Initializer method.
func (db *DB) Initialize() {
	db.initializer(db)
}

// Initializer changes the initialization function run when the Initialize method is called.
func (db *DB) Initializer(initializer func(*DB)) {
	db.initializer = initializer
}

// Ping makes a call to the database and returns an error if any
func (db *DB) Ping() error {
	_, err := db.Client.Exec("SELECT 1")
	return err
}

// RunMigrations is an initialization function for a DB which attempts to run migrations
// once a second in a loop until they run successfully.
func RunMigrations(db *DB) {
	for {
		db.Logger.Debug("DB.RunMigrations: Running migrations.")
		err := db.Migrate()
		if err != nil {
			db.Logger.Errorf("DB.RunMigrations: error %v running migrations, will retry in 1 second.", err)
			time.Sleep(1 * time.Second)
			continue
		}
		db.Logger.Debug("DB.RunMigrations: Migrations finished.")
		break
	}
}
