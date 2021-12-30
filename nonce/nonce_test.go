package nonce

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestClient struct {
	nonce uint64
	step  uint64
}

var _ Client = (*TestClient)(nil)

func (c *TestClient) Nonce(ctx context.Context, privKey string) (nonce uint64, err error) {
	c.nonce += c.step
	return c.nonce, nil
}

func TestIncrement(t *testing.T) {
	var (
		ctx            = context.Background()
		workerNum      = 3
		incrementCount = 3
		wg             sync.WaitGroup
	)

	c := TestClient{}

	n, err := NewNonce(ctx, &c, "", false)
	require.NoError(t, err)

	startNonce, _ := n.Current()

	work := func() {
		defer wg.Done()

		for i := 0; i < incrementCount; i++ {
			_, err := n.Increment()
			require.NoError(t, err)
		}
	}

	for i := 0; i < workerNum; i++ {
		wg.Add(1)
		go work()
	}

	wg.Wait()

	endNonce, _ := n.Current()

	expected := uint64(workerNum*incrementCount) - startNonce
	require.Equal(t, expected, endNonce)
}

func TestIncrementEnsure(t *testing.T) {
	var (
		ctx            = context.Background()
		incrementCount = uint64(3)
		step           = uint64(2)
	)

	c := TestClient{
		nonce: 1,
		step:  step,
	}

	n, err := NewNonce(ctx, &c, "", true)
	require.NoError(t, err)

	startNonce, _ := n.Current()

	for i := uint64(1); i <= incrementCount; i++ {
		current, err := n.Increment()
		require.NoError(t, err)

		expected := startNonce + step*i
		require.Equal(t, expected, current)
		require.Equal(t, expected+1, n.current)
	}

	endNonce, _ := n.Current()

	expected := startNonce + step*(incrementCount+1)
	require.Equal(t, expected, endNonce)
}
