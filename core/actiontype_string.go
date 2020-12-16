// Code generated by "stringer -type ActionType -trimprefix ActionType"; DO NOT EDIT.

package core

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ActionTypeSupply-1]
	_ = x[ActionTypeBorrow-2]
	_ = x[ActionTypeRedeem-3]
	_ = x[ActionTypeRepay-4]
	_ = x[ActionTypeMint-5]
	_ = x[ActionTypePledge-6]
	_ = x[ActionTypeUnpledge-7]
	_ = x[ActionTypeSeizeToken-8]
	_ = x[ActionTypeRedeemTransfer-9]
	_ = x[ActionTypeUnpledgeTransfer-10]
	_ = x[ActionTypeBorrowTransfer-11]
	_ = x[ActionTypeSeizeTokenTransfer-12]
	_ = x[ActionTypeRefundTransfer-13]
	_ = x[ActionTypeRepayRefundTransfer-14]
	_ = x[ActionTypeSeizeRefundTransfer-15]
	_ = x[ActionTypeProposalAddMarket-16]
	_ = x[ActionTypeProposalUpdateMarket-17]
	_ = x[ActionTypeProposalWithdrawReserves-18]
	_ = x[ActionTypeProposalProvidePrice-19]
	_ = x[ActionTypeProposalVote-20]
	_ = x[ActionTypeProposalInjectCTokenForMint-21]
}

const _ActionType_name = "SupplyBorrowRedeemRepayMintPledgeUnpledgeSeizeTokenRedeemTransferUnpledgeTransferBorrowTransferSeizeTokenTransferRefundTransferRepayRefundTransferSeizeRefundTransferProposalAddMarketProposalUpdateMarketProposalWithdrawReservesProposalProvidePriceProposalVoteProposalInjectCTokenForMint"

var _ActionType_index = [...]uint16{0, 6, 12, 18, 23, 27, 33, 41, 51, 65, 81, 95, 113, 127, 146, 165, 182, 202, 226, 246, 258, 285}

func (i ActionType) String() string {
	i -= 1
	if i < 0 || i >= ActionType(len(_ActionType_index)-1) {
		return "ActionType(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _ActionType_name[_ActionType_index[i]:_ActionType_index[i+1]]
}
