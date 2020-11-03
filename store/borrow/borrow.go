package borrow

import (
	"compound/core"
	"context"

	"github.com/fox-one/pkg/store/db"
)

type borrowStore struct {
	db *db.DB
}

// New new borrow store
func New(db *db.DB) core.IBorrowStore {
	return &borrowStore{
		db: db,
	}
}

func init() {
	db.RegisterMigrate(func(db *db.DB) error {
		tx := db.Update().Model(core.Borrow{})
		if err := tx.AutoMigrate(core.Borrow{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *borrowStore) Save(ctx context.Context, borrow *core.Borrow) error {
	if e := s.db.Update().Where("user_id=? and symbol=?", borrow.UserID, borrow.Symbol).FirstOrCreate(borrow).Error; e != nil {
		return e
	}

	return nil
}
func (s *borrowStore) Find(ctx context.Context, userID string, symbol string) (*core.Borrow, error) {
	var borrow core.Borrow
	if e := s.db.View().Where("user_id=? and symbol=?", userID, symbol).First(&borrow).Error; e != nil {
		return nil, e
	}

	return &borrow, nil
}

func (s *borrowStore) FindByUser(ctx context.Context, userID string) ([]*core.Borrow, error) {
	var borrows []*core.Borrow
	if e := s.db.View().Where("user_id=?", userID).Find(&borrows).Error; e != nil {
		return nil, e
	}

	return borrows, nil
}

func (s *borrowStore) Update(ctx context.Context, tx *db.DB, borrow *core.Borrow) error {
	version := borrow.Version
	borrow.Version++
	if err := tx.Update().Model(core.Supply{}).Where("user_id=? and symbol=? and version=?", borrow.UserID, borrow.Symbol, version).Updates(borrow).Error; err != nil {
		return err
	}

	return nil
}
