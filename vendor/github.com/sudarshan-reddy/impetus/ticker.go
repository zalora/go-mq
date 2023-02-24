package impetus

import "time"

// ImmediateTicker is a wrapper a ticker that produces the first tick
// instantaneously because. time.Ticker waits for the time.Duration
// it started with before the first tick.
type ImmediateTicker struct {
	*time.Ticker
}

// NewImmediateTicker creates a new instance of ImmediateTicker by
// wrapping Ticker around it
func NewImmediateTicker(duration time.Duration) *ImmediateTicker {
	ticker := time.NewTicker(duration)
	orgCh := ticker.C
	tickerCh := make(chan time.Time, 1)
	ticker.C = tickerCh
	go func() {
		defer close(tickerCh)
		tickerCh <- time.Now()
		for tick := range orgCh {
			tickerCh <- tick
		}
	}()
	return &ImmediateTicker{Ticker: ticker}
}

// Stop Gracefully closes the wrapped ticker and the wrapping ticker
func (b *ImmediateTicker) Stop() {
	b.Ticker.Stop()
}
