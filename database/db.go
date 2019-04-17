package database

import (
	"github.com/go-pg/pg"
	"github.com/robinjoseph08/go-pg-migrations"
	"log"
	"time"
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
	EnableLogging bool
}

// DB wraps a client for a database.
type DB struct {
	Client *pg.DB
	Logger *log.Logger
}

// Close closes a connection to a database. Once close has been called calling other methods on db will error.
func (db *DB) Close() {
	db.Logger.Println("Closing database connection")
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
	}
	d.logger.Printf("executed query\n%+v ", query)
}

// New returns a new DB object which wraps
// a connection to the database specified in config
func New(config DBConfig, logger *log.Logger) DB {
	db := pg.Connect(&pg.Options{
		Addr:     config.Address,
		User:     config.User,
		Database: config.Database,
		Password: config.Password,
	})
	if config.EnableLogging {
		db.AddQueryHook(dbLogger{logger: logger})
	}
	return DB{
		Client: db,
		Logger: logger,
	}
}

// Initialize starts up the database and returns the Close function used to gracefully shut it down.
func (db *DB) Initialize() func() {
	for {
		err := db.Migrate()
		if err != nil {
			db.Logger.Println(err)
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	return db.Close
}
