package serum

import (
	"fmt"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"
)

var DEX_PROGRAM_ID = solana.MustPublicKeyFromBase58("EUqojwWA2rd19FZrzeBncJsm38Jm1hEhE3zsmX3bRc2o")

func init() {
	solana.RegisterInstructionDecoder(DEX_PROGRAM_ID, registryDecodeInstruction)
}

func registryDecodeInstruction(accounts []solana.PublicKey, rawInstruction *solana.CompiledInstruction) (interface{}, error) {
	inst, err := DecodeInstruction(accounts, rawInstruction)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

func DecodeInstruction(accounts []solana.PublicKey, compiledInstruction *solana.CompiledInstruction) (*Instruction, error) {
	var inst Instruction
	if err := bin.NewDecoder(compiledInstruction.Data).Decode(&inst); err != nil {
		return nil, fmt.Errorf("unable to decode instruction for serum program: %w", err)
	}

	if v, ok := inst.Impl.(solana.AccountSettable); ok {
		err := v.SetAccounts(accounts, compiledInstruction.Accounts)
		if err != nil {
			return nil, fmt.Errorf("unable to set accounts for instruction: %w", err)
		}
	}

	return &inst, nil
}

type Instruction struct {
	bin.BaseVariant
	Version uint8
}

func (i *Instruction) String() string {
	return fmt.Sprintf("%s", i.Impl)
}

func (i *Instruction) UnmarshalBinary(decoder *bin.Decoder) (err error) {
	i.Version, err = decoder.ReadUint8()
	if err != nil {
		return fmt.Errorf("unable to read version: %w", err)
	}
	return i.BaseVariant.UnmarshalBinaryVariant(decoder, InstructionDefVariant)
}

func (i *Instruction) MarshalBinary(encoder *bin.Encoder) error {
	err := encoder.WriteUint8(i.Version)
	if err != nil {
		return fmt.Errorf("unable to write instruction version: %w", err)
	}

	err = encoder.WriteUint32(i.TypeID)
	if err != nil {
		return fmt.Errorf("unable to write variant type: %w", err)
	}
	return encoder.Encode(i.Impl)
}

var InstructionDefVariant = bin.NewVariantDefinition(bin.Uint32TypeIDEncoding, []bin.VariantType{
	{"initialize_market", (*InstructionInitializeMarket)(nil)},
	{"new_order", (*InstructionNewOrder)(nil)},
	{"match_orders", (*InstructionMatchOrder)(nil)},
	{"consume_events", (*InstructionConsumeEvents)(nil)},
	{"cancel_order", (*InstructionCancelOrder)(nil)},
	{"settle_funds", (*InstructionSettleFunds)(nil)},
	{"cancel_order_by_client_id", (*InstructionCancelOrderByClientId)(nil)},
})

type InitializeMarketAccounts struct {
	Market        solana.AccountMeta
	SPLCoinToken  solana.AccountMeta
	SPLPriceToken solana.AccountMeta
	CoinMint      solana.AccountMeta
	PriceMint     solana.AccountMeta
}

type InstructionInitializeMarket struct {
	BaseLotSize        uint64
	QuoteLotSize       uint64
	FeeRateBps         uint16
	VaultSignerNonce   uint64
	QuoteDustThreshold uint64

	Accounts *InitializeMarketAccounts `bin:"-"`
}

func (i *InstructionInitializeMarket) SetAccounts(accounts []solana.PublicKey, instructionActIdx []uint8) error {
	if len(instructionActIdx) < 9 {
		return fmt.Errorf("insuficient account, Initialize Market requires at-least 8 accounts not %d", len(accounts))
	}
	i.Accounts = &InitializeMarketAccounts{
		Market:        solana.AccountMeta{accounts[instructionActIdx[0]], false, true},
		SPLCoinToken:  solana.AccountMeta{accounts[instructionActIdx[5]], false, true},
		SPLPriceToken: solana.AccountMeta{accounts[instructionActIdx[6]], false, true},
		CoinMint:      solana.AccountMeta{accounts[instructionActIdx[7]], false, false},
		PriceMint:     solana.AccountMeta{accounts[instructionActIdx[8]], false, false},
	}
	return nil
}

type NewOrderAccounts struct {
	Market             solana.AccountMeta
	OpenOrders         solana.AccountMeta
	RequestQueue       solana.AccountMeta
	Payer              solana.AccountMeta
	Owner              solana.AccountMeta
	CoinVault          solana.AccountMeta
	PCVault            solana.AccountMeta
	SPLTokenProgram    solana.AccountMeta
	Rent               solana.AccountMeta
	SRMDiscountAccount *solana.AccountMeta
}

func (a *NewOrderAccounts) String() string {
	out := a.Market.String() + " Market\n"
	out += a.OpenOrders.String() + " OpenOrders\n"
	out += a.RequestQueue.String() + " RequestQueue\n"
	out += a.Payer.String() + " Payer\n"
	out += a.Owner.String() + " Owner\n"
	out += a.CoinVault.String() + " CoinVault\n"
	out += a.PCVault.String() + " PCVault\n"
	out += a.SPLTokenProgram.String() + " SPLTokenProgram\n"
	out += a.Rent.String() + " Rent\n"
	if a.SRMDiscountAccount != nil {
		out += a.SRMDiscountAccount.String() + " SRMDiscountAccount"
	}

	return out
}

type InstructionNewOrder struct {
	Side        Side
	LimitPrice  uint64
	MaxQuantity uint64
	OrderType   OrderType
	ClientID    uint64

	Accounts *NewOrderAccounts `bin:"-"`
}

func (i *InstructionNewOrder) String() string {
	out := "New Order\n"
	out += fmt.Sprintf("Side: %q OrderType: %q Limit price: %d  Max Quantity: %d Client ID: %d\n", i.Side, i.OrderType, i.LimitPrice, i.MaxQuantity, i.ClientID)
	out += "Accounts:\n"

	if i.Accounts != nil {
		out += i.Accounts.String()
	}

	return out
}

func (i *InstructionNewOrder) SetAccounts(accounts []solana.PublicKey, instructionActIdx []uint8) error {
	if len(instructionActIdx) < 9 {
		return fmt.Errorf("insuficient account, New Order requires at-least 10 accounts not %d", len(accounts))
	}
	i.Accounts = &NewOrderAccounts{
		Market:          solana.AccountMeta{accounts[instructionActIdx[0]], false, true},
		OpenOrders:      solana.AccountMeta{accounts[instructionActIdx[1]], false, true},
		RequestQueue:    solana.AccountMeta{accounts[instructionActIdx[2]], false, true},
		Payer:           solana.AccountMeta{accounts[instructionActIdx[3]], false, true},
		Owner:           solana.AccountMeta{accounts[instructionActIdx[4]], true, false},
		CoinVault:       solana.AccountMeta{accounts[instructionActIdx[5]], false, true},
		PCVault:         solana.AccountMeta{accounts[instructionActIdx[6]], false, true},
		SPLTokenProgram: solana.AccountMeta{accounts[instructionActIdx[7]], false, false},
		Rent:            solana.AccountMeta{accounts[instructionActIdx[8]], false, false},
	}

	if len(instructionActIdx) >= 10 {
		i.Accounts.SRMDiscountAccount = &solana.AccountMeta{PublicKey: accounts[instructionActIdx[9]], IsWritable: true}
	}

	return nil
}

type MatchOrderAccounts struct {
	Market            solana.AccountMeta
	RequestQueue      solana.AccountMeta
	EventQueue        solana.AccountMeta
	Bids              solana.AccountMeta
	Asks              solana.AccountMeta
	CoinFeeReceivable solana.AccountMeta
	PCFeeReceivable   solana.AccountMeta
}

func (a *MatchOrderAccounts) String() string {
	out := a.Market.String() + " Market\n"
	out += a.RequestQueue.String() + " RequestQueue\n"
	out += a.EventQueue.String() + " EventQueue\n"
	out += a.Bids.String() + " Bids\n"
	out += a.Asks.String() + " Asks\n"
	out += a.CoinFeeReceivable.String() + " CoinFeeReceivable\n"
	out += a.PCFeeReceivable.String() + " PCFeeReceivable"
	return out
}

type InstructionMatchOrder struct {
	Limit uint16

	Accounts *MatchOrderAccounts `bin:"-"`
}

func (i *InstructionMatchOrder) SetAccounts(accounts []solana.PublicKey, instructionActIdx []uint8) error {
	if len(instructionActIdx) < 7 {
		return fmt.Errorf("insuficient account, Match Order requires at-least 7 accounts not %d\n", len(accounts))
	}
	i.Accounts = &MatchOrderAccounts{
		Market:            solana.AccountMeta{PublicKey: accounts[instructionActIdx[0]], IsWritable: true},
		RequestQueue:      solana.AccountMeta{PublicKey: accounts[instructionActIdx[1]], IsWritable: true},
		EventQueue:        solana.AccountMeta{PublicKey: accounts[instructionActIdx[2]], IsWritable: true},
		Bids:              solana.AccountMeta{PublicKey: accounts[instructionActIdx[3]], IsWritable: true},
		Asks:              solana.AccountMeta{PublicKey: accounts[instructionActIdx[4]], IsWritable: true},
		CoinFeeReceivable: solana.AccountMeta{PublicKey: accounts[instructionActIdx[5]], IsWritable: true},
		PCFeeReceivable:   solana.AccountMeta{PublicKey: accounts[instructionActIdx[6]], IsWritable: true},
	}
	return nil
}

func (i *InstructionMatchOrder) String() string {
	out := "Match Order\n"
	out += fmt.Sprintf("Limit: %d\n", i.Limit)
	out += "Accounts:\n"

	if i.Accounts != nil {
		out += i.Accounts.String()
	}

	return out
}

type ConsumeEventsAccounts struct {
	OpenOrders        []solana.AccountMeta
	Market            solana.AccountMeta
	EventQueue        solana.AccountMeta
	CoinFeeReceivable solana.AccountMeta
	PCFeeReceivable   solana.AccountMeta
}

type InstructionConsumeEvents struct {
	Limit uint16

	Accounts *ConsumeEventsAccounts `bin:"-"`
}

func (i *InstructionConsumeEvents) SetAccounts(accounts []solana.PublicKey, instructionActIdx []uint8) error {
	if len(instructionActIdx) < 4 {
		return fmt.Errorf("insuficient account, Consume Events requires at-least 4 accounts not %d", len(accounts))
	}
	i.Accounts = &ConsumeEventsAccounts{
		Market:            solana.AccountMeta{PublicKey: accounts[instructionActIdx[len(instructionActIdx)-4]], IsWritable: true},
		EventQueue:        solana.AccountMeta{PublicKey: accounts[instructionActIdx[len(instructionActIdx)-3]], IsWritable: true},
		CoinFeeReceivable: solana.AccountMeta{PublicKey: accounts[instructionActIdx[len(instructionActIdx)-2]], IsWritable: true},
		PCFeeReceivable:   solana.AccountMeta{PublicKey: accounts[instructionActIdx[len(instructionActIdx)-1]], IsWritable: true},
	}

	for itr := 0; itr < len(instructionActIdx)-4; itr++ {
		i.Accounts.OpenOrders = append(i.Accounts.OpenOrders, solana.AccountMeta{PublicKey: accounts[instructionActIdx[itr]], IsWritable: true})
	}

	return nil
}

type CancelOrderAccounts struct {
	Market       solana.AccountMeta
	OpenOrders   solana.AccountMeta
	RequestQueue solana.AccountMeta
	Owner        solana.AccountMeta
}

func (a *CancelOrderAccounts) String() string {
	out := a.Market.String() + " Market\n"
	out += a.OpenOrders.String() + " OpenOrders\n"
	out += a.RequestQueue.String() + " RequestQueue\n"
	out += a.Owner.String() + " Owner"
	return out
}

type InstructionCancelOrder struct {
	Side          uint32
	OrderID       bin.Uint128
	OpenOrders    solana.PublicKey
	OpenOrderSlot uint8

	Accounts *CancelOrderAccounts `bin:"-"`
}

func (i *InstructionCancelOrder) String() string {
	out := "Cancel Order\n"
	out += fmt.Sprintf("Side: %q Order ID: %q Open Orders: %s Slot: %d\n", i.Side, i.OrderID, i.OpenOrders.String(), i.OpenOrderSlot)
	out += "Accounts:\n"

	if i.Accounts != nil {
		out += i.Accounts.String()
	}

	return out
}

func (i *InstructionCancelOrder) SetAccounts(accounts []solana.PublicKey, instructionActIdx []uint8) error {
	if len(instructionActIdx) < 4 {
		return fmt.Errorf("insuficient account, Cancel Order requires at-least 4 accounts not %d\n", len(accounts))
	}
	i.Accounts = &CancelOrderAccounts{
		Market:       solana.AccountMeta{accounts[instructionActIdx[0]], false, false},
		OpenOrders:   solana.AccountMeta{accounts[instructionActIdx[1]], false, true},
		RequestQueue: solana.AccountMeta{accounts[instructionActIdx[2]], false, true},
		Owner:        solana.AccountMeta{accounts[instructionActIdx[3]], true, false},
	}

	return nil
}

type SettleFundsAccounts struct {
	Market           solana.AccountMeta
	OpenOrders       solana.AccountMeta
	Owner            solana.AccountMeta
	CoinVault        solana.AccountMeta
	PCVault          solana.AccountMeta
	CoinWallet       solana.AccountMeta
	PCWallet         solana.AccountMeta
	Signer           solana.AccountMeta
	SPLTokenProgram  solana.AccountMeta
	ReferrerPCWallet *solana.AccountMeta
}

func (a *SettleFundsAccounts) String() string {
	out := a.Market.String() + " Market\n"
	out += a.OpenOrders.String() + " OpenOrders\n"
	out += a.Owner.String() + " Owner\n"
	out += a.CoinVault.String() + " CoinVault\n"
	out += a.PCVault.String() + " PCVault\n"
	out += a.CoinWallet.String() + " CoinWallet\n"
	out += a.PCWallet.String() + " PCWallet\n"
	out += a.Signer.String() + " Signer\n"
	out += a.SPLTokenProgram.String() + " SPLTokenProgram\n"
	out += a.ReferrerPCWallet.String() + " ReferrerPCWallet"

	return out
}

type InstructionSettleFunds struct {
	Accounts *SettleFundsAccounts `bin:"-"`
}

func (i *InstructionSettleFunds) String() string {
	out := "Settle Funds\n"
	out += "Accounts:\n"
	if i.Accounts != nil {
		out += i.Accounts.String()
	}
	return out
}

func (i *InstructionSettleFunds) SetAccounts(accounts []solana.PublicKey, instructionActIdx []uint8) error {
	if len(instructionActIdx) < 9 {
		return fmt.Errorf("insuficient account, Settle Funds requires at-least 10 accounts not %d", len(accounts))
	}
	i.Accounts = &SettleFundsAccounts{
		Market:          solana.AccountMeta{accounts[instructionActIdx[0]], false, true},
		OpenOrders:      solana.AccountMeta{accounts[instructionActIdx[1]], false, true},
		Owner:           solana.AccountMeta{accounts[instructionActIdx[2]], true, false},
		CoinVault:       solana.AccountMeta{accounts[instructionActIdx[3]], false, true},
		PCVault:         solana.AccountMeta{accounts[instructionActIdx[4]], false, true},
		CoinWallet:      solana.AccountMeta{accounts[instructionActIdx[5]], false, true},
		PCWallet:        solana.AccountMeta{accounts[instructionActIdx[6]], false, true},
		Signer:          solana.AccountMeta{accounts[instructionActIdx[7]], false, false},
		SPLTokenProgram: solana.AccountMeta{accounts[instructionActIdx[8]], false, false},
	}

	if len(instructionActIdx) >= 10 {
		i.Accounts.ReferrerPCWallet = &solana.AccountMeta{PublicKey: accounts[instructionActIdx[9]], IsWritable: true}
	}

	return nil
}

type CancelOrderByClientIdAccounts struct {
	Market       solana.AccountMeta
	OpenOrders   solana.AccountMeta
	RequestQueue solana.AccountMeta
	Owner        solana.AccountMeta
}

func (a *CancelOrderByClientIdAccounts) String() string {
	out := a.Market.String() + " Market\n"
	out += a.OpenOrders.String() + " OpenOrders\n"
	out += a.RequestQueue.String() + " RequestQueue\n"
	out += a.Owner.String() + " Owner"
	return out
}

type InstructionCancelOrderByClientId struct {
	ClientID uint64

	Accounts *CancelOrderByClientIdAccounts
}

func (i *InstructionCancelOrderByClientId) String() string {
	out := "Cancel Order by Client\n"
	out += fmt.Sprintf("ClientID: %d\n", i.ClientID)
	out += "Accounts:\n"

	if i.Accounts != nil {
		out += i.Accounts.String()
	}

	return out
}

func (i *InstructionCancelOrderByClientId) SetAccounts(accounts []solana.PublicKey, instructionActIdx []uint8) error {
	if len(instructionActIdx) < 4 {
		return fmt.Errorf("insuficient account, Cancel Order By Client Id requires at-least 4 accounts not %d", len(accounts))
	}
	i.Accounts = &CancelOrderByClientIdAccounts{
		Market:       solana.AccountMeta{accounts[instructionActIdx[0]], false, false},
		OpenOrders:   solana.AccountMeta{accounts[instructionActIdx[1]], false, true},
		RequestQueue: solana.AccountMeta{accounts[instructionActIdx[2]], false, true},
		Owner:        solana.AccountMeta{accounts[instructionActIdx[3]], true, false},
	}

	return nil
}

type SideType string

const (
	SideLayoutTypeUnknown SideType = "UNKNOWN"
	SideLayoutTypeBid     SideType = "BID"
	SideLayoutTypeAsk     SideType = "ASK"
)

type Side uint32

func (s Side) getSide() SideType {
	switch s {
	case 0:
		return SideLayoutTypeBid
	case 1:
		return SideLayoutTypeAsk
	}
	return SideLayoutTypeUnknown
}

func (s Side) String() string {
	return string(s.getSide())
}

type OrderTypeString string

const (
	OrderTypeUnknown           OrderTypeString = "UNKNOWN"
	OrderTypeLimit             OrderTypeString = "LIMIT"
	OrderTypeImmediateOrCancel OrderTypeString = "IMMEDIATE_OR_CANCEL"
	OrderTypePostOnly          OrderTypeString = "POST_ONLY"
)

type OrderType uint32

func (o OrderType) getOrderType() OrderTypeString {
	switch o {
	case 0:
		return OrderTypeLimit
	case 1:
		return OrderTypeImmediateOrCancel
	case 2:
		return OrderTypePostOnly
	}
	return OrderTypeUnknown
}
func (t OrderType) String() string {
	return string(t.getOrderType())
}