package app

import (
	"context"
	"fmt"
	"os"
	"shpong/config"
	matrix_db "shpong/db/matrix/gen"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

type DB struct {
	*pgxpool.Pool
}

type MatrixDB struct {
	*pgxpool.Pool
	Queries *matrix_db.Queries
}

// NewDB returns a new database instace
func NewMatrixDB() (*MatrixDB, error) {

	c, err := config.Read(CONFIG_FILE)
	if err != nil {
		panic(err)
	}

	address := c.DB.Matrix

	conn, err := pgxpool.New(context.Background(), address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	err = conn.Ping(context.Background())

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	q := matrix_db.New(conn)

	store := &MatrixDB{conn, q}

	return store, nil
}
