package nonce

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"
	"sync/atomic"
)

const (
	DefaultFailedListCap = 255
)

type Nonce struct {
	sync.Mutex

	client  Client
	privKey string

	current uint64
	ensure  bool // check latest nonce in mem pool

	// assign noce number from here first
	failedList []int
	tail       int
}

func NewNonce(ctx context.Context, client Client, privKey string, ensure bool, failedListCap int) (n *Nonce, err error) {
	n = &Nonce{}

	if failedListCap == 0 {
		failedListCap = DefaultFailedListCap
	}

	n.failedList = make([]int, failedListCap)

	// fill all slot
	for i := range n.failedList {
		n.failedList[i] = math.MaxInt
	}

	if n.current, err = client.Nonce(ctx, privKey); err != nil {
		return
	}

	if ensure {
		n.client = client
		n.privKey = privKey
		n.ensure = true
	}

	return
}

func (n *Nonce) Increment() (uint64, error) {
	if !n.ensure {
		return atomic.AddUint64(&n.current, 1) - 1, nil
	}

	n.Lock()
	defer n.Unlock()

	current, err := n.client.Nonce(context.Background(), n.privKey)
	if err != nil {
		return 0, err
	}

	if current > n.current {
		n.current = current
	}

	n.current += 1
	return n.current - 1, nil
}

func (n *Nonce) Decrement() (uint64, error) {
	if !n.ensure {
		return atomic.AddUint64(&n.current, ^uint64(0)), nil
	}

	n.Lock()
	defer n.Unlock()

	n.current -= 1

	current, err := n.client.Nonce(context.Background(), n.privKey)
	if err != nil {
		return 0, err
	}

	if n.current < current {
		n.current = current
	}

	return n.current, nil
}

func (n *Nonce) Reset(nonce uint64) {
	atomic.StoreUint64(&n.current, nonce)
}

func (n *Nonce) Current() (uint64, error) {
	if !n.ensure {
		return atomic.LoadUint64(&n.current), nil
	}

	n.Lock()
	defer n.Unlock()

	current, err := n.client.Nonce(context.Background(), n.privKey)
	if err != nil {
		return 0, err
	}

	if current > n.current {
		return current, nil
	}

	return n.current, nil
}

func (n *Nonce) Assign() (uint64, error) {
	n.Lock()

	if 0 < n.tail {
		defer n.Unlock()
		return n.popFailedNonce()
	}
	n.Unlock()

	return n.Increment()
}

func (n *Nonce) Next() (uint64, error) {
	n.Lock()

	if 0 < n.tail {
		defer n.Unlock()
		return uint64(n.failedList[0]), nil
	}
	n.Unlock()

	return n.Current()
}

func (n *Nonce) AddFailedNonce(nonce uint64) error {
	n.Lock()
	defer n.Unlock()

	if cap(n.failedList) <= n.tail {
		return errors.New("overflow of nonce failed list")
	}

	if math.MaxInt < nonce {
		return errors.New("overflow of nonce value")
	}

	n.failedList[n.tail] = int(nonce)
	n.tail++
	sort.IntSlice(n.failedList).Sort()

	return nil
}

func (n *Nonce) popFailedNonce() (nonce uint64, err error) {
	if n.tail <= 0 {
		err = errors.New("empty failed nonce list")
		return
	}

	nonce = uint64(n.failedList[0])
	n.failedList[0] = math.MaxInt

	// shift up if more than 1
	if 1 < n.tail {
		for i := 0; i < n.tail-1; i++ {
			n.failedList[i], n.failedList[i+1] = n.failedList[i+1], n.failedList[i]
		}
	}

	n.tail--
	if n.tail < 0 {
		panic("unexpected! the tail of nonce failed list is negative")
	}

	return
}
