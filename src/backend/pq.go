package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

const schema = `CREATE TABLE IF NOT EXISTS todo (
   id        BIGINT CONSTRAINT pk PRIMARY KEY,
   title     TEXT,
   completed BOOLEAN,
   orderidx  BIGINT,
   url       TEXT
)`

type PostgreSQLTODOService struct {
	pool *pgxpool.Pool
}

func applySchema(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil
	}
	defer conn.Release()
	_, err = conn.Exec(ctx, schema)
	return err
}

func connectToPool(ctx context.Context, url string) *pgxpool.Pool {
	for {
		pool, err := pgxpool.Connect(context.Background(), url)
		if err == nil {
			return pool
		}
		fmt.Printf("Failed to connect to PostgreSQL, retrying: %s\n", err)
		time.Sleep(time.Second)
	}
}

func NewPostgreSQLTODOService(ctx context.Context, url string) (*PostgreSQLTODOService, error) {
	pool := connectToPool(ctx, url)
	if err := applySchema(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgreSQLTODOService{pool: pool}, nil
}

func (pts *PostgreSQLTODOService) GetAll(ctx context.Context) ([]TODO, error) {
	conn, err := pts.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var ret []TODO
	rows, err := conn.Query(ctx, "SELECT id, title, completed, orderidx, url FROM todo")
	for rows.Next() {
		var todo TODO
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Completed, &todo.Order, &todo.URL); err != nil {
			return nil, err
		}
		ret = append(ret, todo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (pts *PostgreSQLTODOService) Get(ctx context.Context, id int) (*TODO, error) {
	conn, err := pts.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	row := conn.QueryRow(ctx, "SELECT id, title, completed, orderidx, url FROM todo WHERE id = %", id)
	var ret TODO
	if err := row.Scan(&ret.ID, &ret.Title, &ret.Completed, &ret.Order, &ret.URL); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ret, nil
}

func (pts *PostgreSQLTODOService) Save(ctx context.Context, todo TODO) error {
	conn, err := pts.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Query(ctx, `INSERT INTO todo (id, title, completed, orderidx, url)
VALUES (%, %, %, %, %)
ON CONFLICT (id) DO UPDATE`,
		todo.ID, todo.Title, todo.Completed, todo.Order, todo.URL)
	return err
}

func (pts *PostgreSQLTODOService) DeleteAll(ctx context.Context) error {
	conn, err := pts.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Query(ctx, "DELETE FROM todo")
	return err
}

func (pts *PostgreSQLTODOService) Delete(ctx context.Context, id int) error {
	conn, err := pts.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Query(ctx, "DELETE FROM todo WHERE id = %", id)
	return err
}
