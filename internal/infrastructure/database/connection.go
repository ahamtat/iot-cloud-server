package database

import (
	"fmt"

	"github.com/pkg/errors"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type Connection struct {
	dsn string
	Db  *sqlx.DB
}

func NewDatabaseConnection(user, password, host, database string, port int) *Connection {
	c := &Connection{
		dsn: fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?checkConnLiveness=true",
			user, password, host, port, database),
	}
	return c
}

// Create and initialize mysql connection
func (c *Connection) Init() error {
	// Open a connection and validate with ping
	db, err := sqlx.Connect("mysql", c.dsn)
	if err != nil {
		return errors.Wrap(err, "failed to connect to mysql")
	}

	// Set connection pool params
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0) // connections are reused forever

	c.Db = db
	return nil
}

// Close mysql connection
func (c *Connection) Close() error {
	err := c.Db.Close()
	if err != nil {
		return errors.Wrap(err, "failed closing mysql connection")
	}
	return nil
}
