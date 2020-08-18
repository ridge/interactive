package main

import (
	"context"

	"github.com/jackc/pgx/v4"
)

type PostgreSQLTODOService struct {
	conn *pgx.Conn
}

func NewPostgreSQLTODOService(url string) (*PostgreSQLTODOService, error) {
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}
	return &PostgreSQLTODOService{conn: conn}, nil
}

func (pts *PostgreSQLTODOService) GetAll() ([]TODO, error) {
	return nil, nil
}

func (pts *PostgreSQLTODOService) Get(id int) (*TODO, error) {
	return nil, nil
}

func (pts *PostgreSQLTODOService) Save(todo *TODO) error {
	return nil
}

func (pts *PostgreSQLTODOService) DeleteAll() error {
	return nil
}

func (pts *PostgreSQLTODOService) Delete(id int) error {
	return nil
}
