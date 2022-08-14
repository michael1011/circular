package parallel

import (
	"circular/graph"
	"circular/node"
	rebalance2 "circular/rebalance"
	"circular/util"
	"fmt"
	"github.com/elementsproject/glightning/glightning"
	"github.com/elementsproject/glightning/jrpc2"
	"github.com/gammazero/deque"
	"sync"
	"time"
)

type RebalanceParallel struct {
	InScid              string                       `json:"inscid"`
	Amount              uint64                       `json:"amount,omitempty"`
	MaxPPM              uint64                       `json:"maxppm,omitempty"`
	Splits              int                          `json:"splits,omitempty"`
	SplitAmount         uint64                       `json:"splitamount,omitempty"`
	OutList             []string                     `json:"outlist,omitempty"`
	MaxOutPPM           uint64                       `json:"maxoutppm,omitempty"`
	DepleteUpToPercent  float64                      `json:"depleteuptopercent,omitempty"`
	DepleteUpToAmount   uint64                       `json:"depleteuptoamount,omitempty"`
	Attempts            int                          `json:"attempts,omitempty"`
	MaxHops             int                          `json:"maxhops,omitempty"`
	TotalAttempts       uint64                       `json:"-"`
	Node                *node.Node                   `json:"-"`
	InChannel           *graph.Channel               `json:"-"`
	Candidates          *deque.Deque[*graph.Channel] `json:"-"`
	AmountRebalanced    uint64                       `json:"-"`
	InFlightAmount      uint64                       `json:"-"`
	AmountLock          *sync.Mutex                  `json:"-"`
	QueueLock           *sync.Mutex                  `json:"-"`
	RebalanceResultChan chan *rebalance2.Result      `json:"-"`
}

func (r *RebalanceParallel) Name() string {
	return "circular-parallel"
}

func (r *RebalanceParallel) New() interface{} {
	return &RebalanceParallel{}
}

func (r *RebalanceParallel) Call() (jrpc2.Result, error) {
	r.Node = node.GetNode()
	if r.InScid == "" {
		return nil, util.ErrNoRequiredParameter
	}
	r.AmountLock = &sync.Mutex{}
	r.QueueLock = &sync.Mutex{}
	r.TotalAttempts = 0
	r.RebalanceResultChan = make(chan *rebalance2.Result)

	r.setDefaults()

	r.Node.Logf(glightning.Debug, "%+v", r)

	if err := r.validateParameters(); err != nil {
		return nil, err
	}

	incomingChannel, err := r.Node.GetIncomingChannelFromScid(r.InScid)
	if err != nil {
		return nil, err
	}
	r.InChannel = incomingChannel

	if err = r.FindCandidates(r.InChannel.Source); err != nil {
		return nil, err
	}

	r.FireCandidates()
	return r.WaitForResult()
}

func (r *RebalanceParallel) FireCandidates() {
	r.Node.Logln(glightning.Debug, "waiting for AmountLock")
	r.AmountLock.Lock()
	r.Node.Logln(glightning.Debug, "got AmountLock")
	defer r.AmountLock.Unlock()

	carryOn := r.AmountRebalanced+r.InFlightAmount < r.Amount
	splitsInFlight := int(r.InFlightAmount / r.SplitAmount)

	r.Node.Logln(glightning.Debug, "Firing candidates")
	r.Node.Logln(glightning.Debug, "AmountRebalanced: ", r.AmountRebalanced, ", InFlightAmount: ", r.InFlightAmount, ", Total Amount:", r.Amount)
	r.Node.Logln(glightning.Debug, "Carry on: ", carryOn, ", Splits in flight: ", splitsInFlight)
	for carryOn && splitsInFlight < r.Splits {
		candidate, err := r.GetNextCandidate()
		if err != nil {
			// no candidate left
			r.Node.Logln(glightning.Debug, err)
			break
		}
		r.Fire(candidate)

		r.InFlightAmount += r.SplitAmount
		carryOn = r.AmountRebalanced+r.InFlightAmount < r.Amount
		splitsInFlight = int(r.InFlightAmount / r.SplitAmount)

		r.Node.Logln(glightning.Debug, "Carry on: ", carryOn, ", Splits in flight: ", splitsInFlight)
	}
}

func (r *RebalanceParallel) WaitForResult() (jrpc2.Result, error) {
	start := time.Now()
	result := NewResult(r.Amount)

	// while there's something inflight, wait for results
	for r.InFlightAmount > 0 {
		r.Node.Logln(glightning.Debug, "Waiting for result, InFlightAmount:", r.InFlightAmount)
		rebalanceResult := <-r.RebalanceResultChan

		r.TotalAttempts += rebalanceResult.Attempts

		if rebalanceResult.Status == "success" {
			r.Node.Logf(glightning.Info, "Successful rebalance: %+v", rebalanceResult)

			// update results data
			result.AddSuccess(rebalanceResult, r.Node.Graph.Aliases)

			// put the candidate back in front of the queue
			scid := rebalanceResult.Route.Hops[0].ShortChannelId
			r.EnqueueCandidate(scid)
		} else {
			r.Node.Logf(glightning.Debug, "Failed rebalance: %+v", rebalanceResult)
		}

		// update inflight and rebalanced amount
		r.UpdateAmounts(rebalanceResult)

		// now that we had a result, we can fire more candidates
		r.FireCandidates()
	}

	// rebalance is over
	result.Attempts = r.TotalAttempts
	result.Time = fmt.Sprintf("%.3fs", float64(time.Since(start).Milliseconds())/1000)
	return result, nil
}

func (r *RebalanceParallel) Fire(candidate *graph.Channel) {
	r.Node.Logln(glightning.Debug, "Firing candidate:", candidate.ShortChannelId)
	rebalance := rebalance2.NewRebalance(candidate, r.InChannel, r.SplitAmount, r.MaxPPM, r.Attempts, r.MaxHops)

	go func() {
		r.RebalanceResultChan <- rebalance.Run()
	}()
}

func (r *RebalanceParallel) UpdateAmounts(result *rebalance2.Result) {
	r.Node.Logln(glightning.Debug, "waiting for AmountLock")
	r.AmountLock.Lock()
	r.Node.Logln(glightning.Debug, "got AmountLock")
	defer r.AmountLock.Unlock()

	r.InFlightAmount -= r.SplitAmount
	if result.Status == "success" {
		r.AmountRebalanced += r.SplitAmount

		// not really a good way to do it, but we need to do this to make sure we don't
		// overshoot the Deplete amount. This is necessary because otherwise the
		// spendable balance would only be updated on refreshPeers.
		scid := result.Route.Hops[0].ShortChannelId
		r.Node.UpdateChannelBalance(result.Out, scid, result.Amount)
	}
}
