package main

import (
	"context"
	"fmt"

	"github.com/tak1827/nonce-incrementor/nonce"
)

type SampleClient struct{}

var (
	_ nonce.Client = (*SampleClient)(nil)

	clientNonce = uint64(0)
)

func (c *SampleClient) Nonce(ctx context.Context, privKey string) (nonce uint64, err error) {
	return clientNonce, nil
}

func main() {
	c := SampleClient{}

	// don't check latest nonce in mempool when increment and get current
	n, err := nonce.NewNonce(context.Background(), &c, "", false)
	if err != nil {
		panic(err)
	}

	n.Increment()
	current, _ := n.Current()
	fmt.Printf("current: %d\n", current)

	// check latest nonce in mempool when increment and get current
	n, err = nonce.NewNonce(context.Background(), &c, "", true)
	if err != nil {
		panic(err)
	}

	n.Increment()
	clientNonce = 100
	current, _ = n.Current()

	fmt.Printf("current: %d\n", current)
}
