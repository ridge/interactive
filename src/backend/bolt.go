package main

import (
	"context"
	"encoding/json"
	"fmt"

	"go.etcd.io/bbolt"
)

var todoBucket = []byte("todo")

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

func (bts *BoltDBTODOService) GetAll(_ context.Context) ([]TODO, error) {
	var ret []TODO
	err := bts.db.View(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(todoBucket)
		if err != nil {
			return err
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var todo TODO
			if err := json.Unmarshal(v, &todo); err != nil {
				return err
			}
			ret = append(ret, todo)
		}
		return nil
	})
	return ret, err
}

func (bts *BoltDBTODOService) Get(_ context.Context, id int) (*TODO, error) {
	var todo *TODO
	err := bts.db.View(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(todoBucket)
		if err != nil {
			return err
		}
		k := fmt.Sprintf("%020d", id)
		v := b.Get([]byte(k))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, todo)
	})
	return todo, err
}

func (bts *BoltDBTODOService) Save(_ context.Context, todo TODO) error {
	return bts.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(todoBucket)
		if err != nil {
			return err
		}
		k := fmt.Sprintf("%020d", todo.ID)
		v, err := json.Marshal(todo)
		if err != nil {
			return err
		}
		return b.Put([]byte(k), v)
	})
}

func (bts *BoltDBTODOService) DeleteAll(_ context.Context) error {
	return bts.db.Update(func(tx *bbolt.Tx) error {
		err := tx.DeleteBucket(todoBucket)
		if err == bbolt.ErrBucketNotFound {
			return nil
		}
		return err
	})
}

func (bts *BoltDBTODOService) Delete(_ context.Context, id int) error {
	return bts.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(todoBucket)
		if err != nil {
			return err
		}
		k := fmt.Sprintf("%020d", id)
		return b.Delete([]byte(k))
	})
}
