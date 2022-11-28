package parallel

import (
	"circular/graph"
	"circular/node"
	"circular/types"
	"github.com/elementsproject/glightning/glightning"
	"github.com/gammazero/deque"
	"sync"
)

type RebalanceMethods interface {
	IsGoodCandidate(peerChannel *glightning.PeerChannel) bool
	CanUseChannel(channel *glightning.PeerChannel) error
	Fire(candidate *graph.Channel)
	EnqueueCandidate(result *types.Result)
	GetCandidateDirection(id string) string
	AddSuccess(result *types.Result)
}

type AbstractRebalance struct {
	TotalAttempts       uint64
	Node                *node.Node
	TargetChannel       *graph.Channel
	Candidates          *deque.Deque[*graph.Channel]
	AmountRebalanced    uint64
	InFlightAmount      uint64
	AmountLock          *sync.Mutex
	QueueLock           *sync.Mutex
	RebalanceResultChan chan *types.Result
	CandidatesList      []string
	Result              *Result
	amount              uint64
	maxPPM              uint64
	splits              int
	splitAmount         uint64
	attempts            int
	maxHops             int
	RebalanceMethods
}

func (r *AbstractRebalance) Init(amount, maxppm, splitamount uint64, splits, attempts, maxhops int) {
	r.Node = node.GetNode()
	r.AmountLock = &sync.Mutex{}
	r.QueueLock = &sync.Mutex{}
	r.TotalAttempts = 0
	r.RebalanceResultChan = make(chan *types.Result)
	r.Node.Logf(glightning.Debug, "%+v", r)
	r.amount = amount
	r.maxPPM = maxppm
	r.splitAmount = splitamount
	r.splits = splits
	r.attempts = attempts
	r.maxHops = maxhops
	r.setGenericDefaults()
	r.Node.Logln(glightning.Debug, "AbstractRebalance initialized")
}
