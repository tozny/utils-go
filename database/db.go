package database

import (
	"crypto/tls"
	"log"
	"time"

	"github.com/go-pg/pg"
	migrations "github.com/robinjoseph08/go-pg-migrations"
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

// dbLogger implements the DBLogger pattern for the go-pg module
// https://github.com/go-pg/pg/wiki/FAQ#how-can-i-viewlog-queries-this-library-generates
type dbLogger struct {
	logger *log.Logger
}

// BeforeQuery is a function that will be invoked
// before any database query is run with the query to run.
func (d dbLogger) BeforeQuery(q *pg.QueryEvent) {}

// AfterQuery is a function that will be executed
// after any database query is run with the query ran.
func (d dbLogger) AfterQuery(q *pg.QueryEvent) {
	query, err := q.FormattedQuery()
	if err != nil {
		d.logger.Printf("error %s executing query\n%+v ", err, query)
		return
	}
	d.logger.Printf("executed query\n%+v ", query)
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
		options.TLSConfig = &tls.Config{}
	}

	db := pg.Connect(options)
	if config.EnableLogging {
		db.AddQueryHook(dbLogger{logger: config.Logger.Raw()})
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
