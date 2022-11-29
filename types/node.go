package types

import "github.com/elementsproject/glightning/glightning"

type Peer struct {
	Id           string            `json:"id"`
	Connected    bool              `json:"connected"`
	NetAddresses []string          `json:"netaddr"`
	Features     *glightning.Hexed `json:"features"`
	Channels     []*PeerChannel    `json:"channels"`
	Logs         []*glightning.Log `json:"log,omitempty"`
}

// PeerChannel we have to define ourselves, because the glightning type is outdated and doesn't have the fee fields
type PeerChannel struct {
	State                            string             `json:"state"`
	ScratchTxId                      string             `json:"scratch_txid"`
	Owner                            string             `json:"owner"`
	ShortChannelId                   string             `json:"short_channel_id"`
	ChannelDirection                 int                `json:"direction"`
	ChannelId                        string             `json:"channel_id"`
	FundingTxId                      string             `json:"funding_txid"`
	CloseToAddress                   string             `json:"close_to_addr,omitempty"`
	CloseToScript                    string             `json:"close_to,omitempty"`
	Status                           []string           `json:"status"`
	Private                          bool               `json:"private"`
	FundingAllocations               map[string]uint64  `json:"funding_allocation_msat"`
	FundingMsat                      map[string]string  `json:"funding_msat"`
	MilliSatoshiToUs                 uint64             `json:"msatoshi_to_us"`
	ToUsMsat                         string             `json:"to_us_msat"`
	MilliSatoshiToUsMin              uint64             `json:"msatoshi_to_us_min"`
	MinToUsMsat                      string             `json:"min_to_us_msat"`
	MilliSatoshiToUsMax              uint64             `json:"msatoshi_to_us_max"`
	MaxToUsMsat                      string             `json:"max_to_us_msat"`
	MilliSatoshiTotal                uint64             `json:"msatoshi_total"`
	TotalMsat                        string             `json:"total_msat"`
	DustLimitSatoshi                 uint64             `json:"dust_limit_satoshis"`
	DustLimitMsat                    string             `json:"dust_limit_msat"`
	MaxHtlcValueInFlightMilliSatoshi uint64             `json:"max_htlc_value_in_flight_msat"`
	MaxHtlcValueInFlightMsat         string             `json:"max_total_htlc_in_msat"`
	TheirChannelReserveSatoshi       uint64             `json:"their_channel_reserve_satoshis"`
	TheirReserveMsat                 string             `json:"their_reserve_msat"`
	OurChannelReserveSatoshi         uint64             `json:"our_channel_reserve_satoshis"`
	OurReserveMsat                   string             `json:"our_reserve_msat"`
	SpendableMilliSatoshi            uint64             `json:"spendable_msatoshi"`
	SpendableMsat                    string             `json:"spendable_msat"`
	ReceivableMilliSatoshi           uint64             `json:"receivable_msatoshi"`
	ReceivableMsat                   string             `json:"receivable_msat"`
	HtlcMinMilliSatoshi              uint64             `json:"htlc_minimum_msat"`
	MinimumHtlcInMsat                string             `json:"minimum_htlc_in_msat"`
	TheirToSelfDelay                 uint               `json:"their_to_self_delay"`
	OurToSelfDelay                   uint               `json:"our_to_self_delay"`
	MaxAcceptedHtlcs                 uint               `json:"max_accepted_htlcs"`
	InPaymentsOffered                uint64             `json:"in_payments_offered"`
	InMilliSatoshiOffered            uint64             `json:"in_msatoshi_offered"`
	IncomingOfferedMsat              string             `json:"in_offered_msat"`
	InPaymentsFulfilled              uint64             `json:"in_payments_fulfilled"`
	InMilliSatoshiFulfilled          uint64             `json:"in_msatoshi_fulfilled"`
	IncomingFulfilledMsat            string             `json:"in_fulfilled_msat"`
	OutPaymentsOffered               uint64             `json:"out_payments_offered"`
	OutMilliSatoshiOffered           uint64             `json:"out_msatoshi_offered"`
	OutgoingOfferedMsat              string             `json:"out_offered_msat"`
	OutPaymentsFulfilled             uint64             `json:"out_payments_fulfilled"`
	OutMilliSatoshiFulfilled         uint64             `json:"out_msatoshi_fulfilled"`
	OutgoingFulfilledMsat            string             `json:"out_fulfilled_msat"`
	FeeBaseMsat                      string             `json:"fee_base_msat"`
	FeeProportionalMillionths        uint64             `json:"fee_proportional_millionths"`
	Htlcs                            []*glightning.Htlc `json:"htlcs"`
}
