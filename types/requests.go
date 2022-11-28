package types

type RebalanceByScid struct {
	OutScid  string `json:"outscid"`
	InScid   string `json:"inscid"`
	Amount   uint64 `json:"amount,omitempty"`
	MaxPPM   uint64 `json:"maxppm,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
	MaxHops  int    `json:"maxhops,omitempty"`
}

func (r RebalanceByScid) Name() string {
	return "circular"
}

type Stop struct {
	Message string `json:"message"`
}

func (s Stop) Name() string {
	return "circular-stop"
}

type Resume struct {
	Message string `json:"message"`
}

func (s Resume) Name() string {
	return "circular-resume"
}
