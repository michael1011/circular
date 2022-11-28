package types

type RebalanceByScid struct {
	OutScid  string `json:"outscid"`
	InScid   string `json:"inscid"`
	Amount   uint64 `json:"amount,omitempty"`
	MaxPPM   uint64 `json:"maxppm,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
	MaxHops  int    `json:"maxhops,omitempty"`
}

func (r *RebalanceByScid) Name() string {
	return "circular"
}
