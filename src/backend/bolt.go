package main

import "go.etcd.io/bbolt"

type BoltDBTODOService struct {
	db *bbolt.DB
}

func NewBoltDBTODOService(file string) (*BoltDBTODOService, error) {
	db, err := bbolt.Open(file, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &BoltDBTODOService{db: db}, nil
}

func (bts *BoltDBTODOService) GetAll() ([]TODO, error) {
	return nil, nil
}

func (bts *BoltDBTODOService) Get(id int) (*TODO, error) {
	return nil, nil
}

func (bts *BoltDBTODOService) Save(todo *TODO) error {
	return nil
}

func (bts *BoltDBTODOService) DeleteAll() error {
	return nil
}

func (bts *BoltDBTODOService) Delete(id int) error {
	return nil
}
