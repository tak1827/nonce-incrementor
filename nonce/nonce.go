package nonce

import (
	"context"
	"sync"
	"sync/atomic"
)

type Nonce struct {
	sync.Mutex

	client  Client
	privKey string

	current uint64
	ensure  bool // check latest nonce in mem pool
}

func NewNonce(ctx context.Context, client Client, privKey string, ensure bool) (n *Nonce, err error) {
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
