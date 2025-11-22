package cache

import (
	"math"
	"sync"
	"time"
)

type blpopQueue struct {
	q  map[string][]chan struct{}
	mu *sync.RWMutex
}

func (bq *blpopQueue) notify(key string) {
	bq.mu.Lock()
	defer bq.mu.Unlock()
	if v, ok := bq.q[key]; !ok || len(v) == 0 {
		return
	} else {
		ch := v[0]
		close(ch)
		bq.q[key] = v[1:]
	}
}

func (bq *blpopQueue) set(key string, out chan struct{}) {
	bq.setWithTime(key, out, time.Second*time.Duration(math.MaxInt32))
}

func (bq *blpopQueue) setWithTime(key string, out chan struct{}, seconds time.Duration) {
	bq.mu.Lock()
	if _, ok := bq.q[key]; !ok {
		bq.q[key] = make([]chan struct{}, 0)
	}
	ch := make(chan struct{})
	bq.q[key] = append(bq.q[key], ch)
	bq.mu.Unlock()

	go func(ch, out chan struct{}, t time.Duration) {
		tick := time.NewTicker(t)
		select {
		case <-ch:
			out <- struct{}{}
		case <-tick.C:
			close(out)
		}
	}(ch, out, seconds)
}
