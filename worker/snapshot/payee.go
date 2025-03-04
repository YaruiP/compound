package snapshot

import (
	"compound/core"
	"compound/pkg/mtg"
	"compound/worker"
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/fox-one/pkg/logger"
	"github.com/fox-one/pkg/property"
	"github.com/fox-one/pkg/store/db"
	uuidutil "github.com/fox-one/pkg/uuid"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

const (
	checkpointKey = "outputs_checkpoint"
	limit         = 500
)

// Payee payee worker
type Payee struct {
	worker.TickWorker
	db                 *db.DB
	system             *core.System
	dapp               *core.Wallet
	propertyStore      property.Store
	userStore          core.UserStore
	outputArchiveStore core.OutputArchiveStore
	walletStore        core.WalletStore
	priceStore         core.IPriceStore
	marketStore        core.IMarketStore
	supplyStore        core.ISupplyStore
	borrowStore        core.IBorrowStore
	proposalStore      core.ProposalStore
	transactionStore   core.TransactionStore
	proposalService    core.ProposalService
	blockService       core.IBlockService
	priceService       core.IPriceOracleService
	marketService      core.IMarketService
	supplyService      core.ISupplyService
	borrowService      core.IBorrowService
	accountService     core.IAccountService
	allowListService   core.IAllowListService
}

// NewPayee new payee
func NewPayee(db *db.DB,
	system *core.System,
	dapp *core.Wallet,
	propertyStore property.Store,
	userStore core.UserStore,
	outputArchiveStore core.OutputArchiveStore,
	walletStore core.WalletStore,
	priceStore core.IPriceStore,
	marketStore core.IMarketStore,
	supplyStore core.ISupplyStore,
	borrowStore core.IBorrowStore,
	proposalStore core.ProposalStore,
	transactionStore core.TransactionStore,
	proposalService core.ProposalService,
	priceSrv core.IPriceOracleService,
	blockService core.IBlockService,
	marketSrv core.IMarketService,
	supplyService core.ISupplyService,
	borrowService core.IBorrowService,
	accountService core.IAccountService,
	allowListService core.IAllowListService) *Payee {
	payee := Payee{
		db:                 db,
		system:             system,
		dapp:               dapp,
		propertyStore:      propertyStore,
		userStore:          userStore,
		outputArchiveStore: outputArchiveStore,
		walletStore:        walletStore,
		priceStore:         priceStore,
		marketStore:        marketStore,
		supplyStore:        supplyStore,
		borrowStore:        borrowStore,
		proposalStore:      proposalStore,
		transactionStore:   transactionStore,
		proposalService:    proposalService,
		priceService:       priceSrv,
		blockService:       blockService,
		marketService:      marketSrv,
		supplyService:      supplyService,
		borrowService:      borrowService,
		accountService:     accountService,
		allowListService:   allowListService,
	}

	return &payee
}

// Run run worker
func (w *Payee) Run(ctx context.Context) error {
	return w.StartTick(ctx, func(ctx context.Context) error {
		return w.onWork(ctx)
	})
}

func (w *Payee) onWork(ctx context.Context) error {
	log := logger.FromContext(ctx).WithField("worker", "payee")

	v, err := w.propertyStore.Get(ctx, checkpointKey)
	if err != nil {
		log.WithError(err).Errorln("property.Get error")
		return err
	}

	outputs, err := w.walletStore.List(ctx, v.Int64(), limit)
	if err != nil {
		log.WithError(err).Errorln("walletStore.List")
		return err
	}

	if len(outputs) <= 0 {
		return errors.New("no more outputs")
	}

	for _, u := range outputs {
		// process the output only once
		_, err := w.outputArchiveStore.Find(ctx, u.TraceID)
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				err = w.db.Tx(func(tx *db.DB) error {
					if err := w.handleOutput(ctx, tx, u); err != nil {
						return err
					}

					//archive output
					archive := core.OutputArchive{
						ID:      u.ID,
						TraceID: u.TraceID,
					}
					// save the processed output
					if err := w.outputArchiveStore.Save(ctx, tx, &archive); err != nil {
						return err
					}
					return nil
				})

				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		if err := w.propertyStore.Save(ctx, checkpointKey, u.ID); err != nil {
			log.WithError(err).Errorln("property.Save:", u.ID)
			return err
		}
	}

	return nil
}

func (w *Payee) handleOutput(ctx context.Context, tx *db.DB, output *core.Output) error {
	log := logger.FromContext(ctx).WithField("output", output.TraceID)
	ctx = logger.WithContext(ctx, log)

	message := w.decodeMemo(output.Memo)

	// handle member vote action
	if member, body, err := core.DecodeMemberProposalTransactionAction(message, w.system.Members); err == nil {
		return w.handleProposalAction(ctx, output, member, body)
	}

	// handle user action
	actionType, body, err := core.DecodeUserTransactionAction(w.system.PrivateKey, message)
	if err != nil {
		log.WithError(err).Errorln("DecodeTransactionAction error")
		return nil
	}

	var reserveUserID uuid.UUID
	// transaction trace id, different from output trace id
	var followID uuid.UUID
	body, err = mtg.Scan(body, &reserveUserID, &followID)
	if err != nil {
		log.WithError(err).Errorln("scan userID and followID error")
		return nil
	}

	userID := output.Sender
	if userID == "" {
		userID = reserveUserID.String()
	}

	//upsert user
	user := core.User{
		UserID:  userID,
		Address: core.BuildUserAddress(userID),
	}
	if err = w.userStore.Save(ctx, &user); err != nil {
		return err
	}

	return w.handleUserAction(ctx, tx, output, actionType, userID, followID.String(), body)
}

func (w *Payee) handleProposalAction(ctx context.Context, output *core.Output, member *core.Member, body []byte) error {
	log := logger.FromContext(ctx)

	var traceID uuid.UUID
	var actionType int

	body, err := mtg.Scan(body, &traceID, &actionType)
	if err != nil {
		log.WithError(err).Debugln("scan proposal trace & action failed")
		return nil
	}

	if core.ActionType(actionType) == core.ActionTypeProposalVote {
		return w.handleVoteProposalEvent(ctx, output, member, traceID.String())
	} else if core.ActionType(actionType) == core.ActionTypeProposalProvidePrice {
		return w.handleProposalProvidePriceEvent(ctx, output, member, traceID.String(), body)
	}

	return w.handleCreateProposalEvent(ctx, output, member, core.ActionType(actionType), traceID.String(), body)
}

func (w *Payee) handleUserAction(ctx context.Context, tx *db.DB, output *core.Output, actionType core.ActionType, userID, followID string, body []byte) error {
	switch actionType {
	case core.ActionTypeSupply:
		return w.handleSupplyEvent(ctx, tx, output, userID, followID, body)
	case core.ActionTypeBorrow:
		return w.handleBorrowEvent(ctx, tx, output, userID, followID, body)
	case core.ActionTypeRedeem:
		return w.handleRedeemEvent(ctx, tx, output, userID, followID, body)
	case core.ActionTypeRepay:
		return w.handleRepayEvent(ctx, tx, output, userID, followID, body)
	case core.ActionTypePledge:
		return w.handlePledgeEvent(ctx, tx, output, userID, followID, body)
	case core.ActionTypeUnpledge:
		return w.handleUnpledgeEvent(ctx, tx, output, userID, followID, body)
	case core.ActionTypeLiquidate:
		return w.handleLiquidationEvent(ctx, tx, output, userID, followID, body)
	default:
		return w.handleRefundEvent(ctx, tx, output, userID, followID, core.ActionTypeRefundTransfer, core.ErrUnknown, "")
	}

}

func (w *Payee) transferOut(ctx context.Context, tx *db.DB, userID, followID, outputTraceID, assetID string, amount decimal.Decimal, transferAction *core.TransferAction) error {
	memoStr, e := transferAction.Format()
	if e != nil {
		return e
	}

	modifier := fmt.Sprintf("%s.%d", followID, transferAction.Source)
	transfer := core.Transfer{
		TraceID:   uuidutil.Modify(outputTraceID, modifier),
		Opponents: []string{userID},
		Threshold: 1,
		AssetID:   assetID,
		Amount:    amount,
		Memo:      memoStr,
	}

	if err := w.walletStore.CreateTransfers(ctx, tx, []*core.Transfer{&transfer}); err != nil {
		logger.FromContext(ctx).WithError(err).Errorln("wallets.CreateTransfers")
		return err
	}

	return nil
}

func (w *Payee) decodeMemo(memo string) []byte {
	if b, err := base64.StdEncoding.DecodeString(memo); err == nil {
		return b
	}

	if b, err := base64.URLEncoding.DecodeString(memo); err == nil {
		return b
	}

	return []byte(memo)
}
