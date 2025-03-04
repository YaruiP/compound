package market

import (
	"compound/core"
	"context"
	"errors"

	"github.com/fox-one/pkg/store/db"
	"github.com/jinzhu/gorm"
)

type marketStore struct {
	db *db.DB
}

// New new market store
func New(db *db.DB) core.IMarketStore {
	return &marketStore{db: db}
}

func init() {
	db.RegisterMigrate(func(db *db.DB) error {
		tx := db.Update().Model(core.Market{})
		if err := tx.AutoMigrate(core.Market{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *marketStore) Save(ctx context.Context, tx *db.DB, market *core.Market) error {
	return tx.Update().Where("asset_id=?", market.AssetID).Create(market).Error
}
func (s *marketStore) Find(ctx context.Context, assetID string) (*core.Market, bool, error) {
	if assetID == "" {
		return nil, true, errors.New("invalid asset_id")
	}

	var market core.Market
	if err := s.db.View().Where("asset_id=?", assetID).First(&market).Error; err != nil {
		return nil, gorm.IsRecordNotFoundError(err), err
	}

	return &market, false, nil
}

func (s *marketStore) FindBySymbol(ctx context.Context, symbol string) (*core.Market, bool, error) {
	if symbol == "" {
		return nil, true, errors.New("invalid symbol")
	}

	var market core.Market
	if err := s.db.View().Where("symbol=?", symbol).First(&market).Error; err != nil {
		return nil, gorm.IsRecordNotFoundError(err), err
	}

	return &market, false, nil
}

func (s *marketStore) FindByCToken(ctx context.Context, ctokenAssetID string) (*core.Market, bool, error) {
	if ctokenAssetID == "" {
		return nil, true, errors.New("invalid ctoken_asset_id")
	}

	var market core.Market
	if err := s.db.View().Where("c_token_asset_id=?", ctokenAssetID).First(&market).Error; err != nil {
		return nil, gorm.IsRecordNotFoundError(err), err
	}

	return &market, false, nil
}

func (s *marketStore) All(ctx context.Context) ([]*core.Market, error) {
	var markets []*core.Market
	if err := s.db.View().Find(&markets).Error; err != nil {
		return nil, err
	}
	return markets, nil
}

func (s *marketStore) AllAsMap(ctx context.Context) (map[string]*core.Market, error) {
	markets, e := s.All(ctx)
	if e != nil {
		return nil, e
	}

	maps := make(map[string]*core.Market)

	for _, m := range markets {
		maps[m.Symbol] = m
	}

	return maps, nil
}

func (s *marketStore) Update(ctx context.Context, tx *db.DB, market *core.Market) error {
	version := market.Version
	market.Version++
	if err := tx.Update().Model(core.Market{}).Where("asset_id=? and version=?", market.AssetID, version).Updates(market).Error; err != nil {
		return err
	}

	return nil
}
