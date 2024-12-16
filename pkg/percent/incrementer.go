package percent

import "github.com/jedib0t/go-pretty/v6/progress"

type Incrementer struct {
	DoneIncrements int
	MaxIncrements  int
}

type IncrementTracker struct {
	Tracker     *progress.Tracker
	Incrementer *Incrementer
}

func (it *IncrementTracker) IncrementSection(err error) {
	if int64(it.Incrementer.DoneIncrements)+int64(1) > int64(it.Incrementer.MaxIncrements) {
		return
	}
	it.Incrementer.DoneIncrements += 1
	if err == nil {
		it.Tracker.Increment(1)
	} else {
		it.Tracker.IncrementWithError(1)
	}
}

func (it *IncrementTracker) CurrentStep() int {
	return it.Incrementer.DoneIncrements
}
