package snapshot

import (
	"compound/core"
	"compound/pkg/id"
	"context"
	"fmt"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/fox-one/pkg/store/db"
	"github.com/shopspring/decimal"
)

// from user
var handleSupplyRedeemEvent = func(ctx context.Context, w *Worker, action core.Action, snapshot *core.Snapshot) error {
	ctokenAssetID := snapshot.AssetID
	market, e := w.marketStore.FindByCToken(ctx, ctokenAssetID)
	if e != nil {
		return handleRefundEvent(ctx, w, action, snapshot)
	}

	redeemTokens := snapshot.Amount.Abs()

	// check redeem allowed
	allowed := w.supplyService.RedeemAllowed(ctx, redeemTokens, market)
	if !allowed {
		return handleRefundEvent(ctx, w, action, snapshot)
	}

	// transfer asset to user
	exchangeRate, e := w.marketService.CurExchangeRate(ctx, market)
	if e != nil {
		return e
	}
	amount := redeemTokens.Mul(exchangeRate)
	trace := id.UUIDFromString(fmt.Sprintf("redeem:%s", snapshot.TraceID))
	input := mixin.TransferInput{
		AssetID:    market.AssetID,
		OpponentID: snapshot.OpponentID,
		Amount:     amount,
		TraceID:    trace,
	}

	if !w.walletService.VerifyPayment(ctx, &input) {
		memo := make(core.Action)
		memo[core.ActionKeyService] = core.ActionServiceRedeemTransfer
		memo[core.ActionKeyCToken] = snapshot.Amount.Abs().String()
		memoStr, e := memo.Format()
		if e != nil {
			return e
		}

		input.Memo = memoStr
		_, e = w.mainWallet.Client.Transfer(ctx, &input, w.mainWallet.Pin)

		if e != nil {
			return e
		}
	}

	return nil
}

//redeem transfer callback, to user
var handleRedeemTransferEvent = func(ctx context.Context, w *Worker, action core.Action, snapshot *core.Snapshot) error {
	assetID := snapshot.AssetID

	reducedCtokens, e := decimal.NewFromString(action[core.ActionKeyCToken])
	if e != nil {
		return e
	}

	return w.db.Tx(func(tx *db.DB) error {
		//TODO update market ctokens, blockNum
		market, e := w.marketStore.Find(ctx, assetID)
		if e != nil {
			return e
		}
		market.CTokens = market.CTokens.Sub(reducedCtokens)
		e = w.marketStore.Update(ctx, tx, market)
		if e != nil {
			return e
		}

		//TODO 更新当前利率，兑换率，使用率

		return nil
	})
}
