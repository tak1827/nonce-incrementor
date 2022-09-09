package nonce

import (
	"context"
	"math"
	"sync"
	"testing"

	// "github.com/davecgh/go-spew/spew"
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

	n, err := NewNonce(ctx, &c, "", false, 0)
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

	n, err := NewNonce(ctx, &c, "", true, 0)
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

func TestFailedList(t *testing.T) {
	var (
		ctx = context.Background()
		c   = TestClient{}
	)

	n, err := NewNonce(ctx, &c, "", true, 5)
	require.NoError(t, err)

	// check the inital state
	require.EqualValues(t, []int{math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt}, n.failedList)

	// check to be sorted
	n.AddFailedNonce(4)
	n.AddFailedNonce(3)
	n.AddFailedNonce(1)
	n.AddFailedNonce(2)
	require.EqualValues(t, []int{1, 2, 3, 4, math.MaxInt}, n.failedList)

	// check overflow error
	n.AddFailedNonce(5)
	require.Equal(t, 5, n.tail)
	require.Error(t, n.AddFailedNonce(6))

	// check pop and sorted
	nonce, _ := n.popFailedNonce()
	require.Equal(t, uint64(1), nonce)
	require.Equal(t, 4, n.tail)
	require.EqualValues(t, []int{2, 3, 4, 5, math.MaxInt}, n.failedList)

	// check can pop all
	nonce, _ = n.popFailedNonce()
	require.Equal(t, uint64(2), nonce)
	require.Equal(t, 3, n.tail)
	nonce, _ = n.popFailedNonce()
	require.Equal(t, uint64(3), nonce)
	require.Equal(t, 2, n.tail)
	nonce, _ = n.popFailedNonce()
	require.Equal(t, uint64(4), nonce)
	require.Equal(t, 1, n.tail)
	nonce, _ = n.popFailedNonce()
	require.Equal(t, uint64(5), nonce)
	require.Equal(t, 0, n.tail)
	require.EqualValues(t, []int{math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt}, n.failedList)

	// check underflow
	_, err = n.popFailedNonce()
	require.Error(t, err)
}

func TestAssignNonce(t *testing.T) {
	var (
		ctx            = context.Background()
		c              = TestClient{}
		workerNum      = 3
		incrementCount = 3
		wg             sync.WaitGroup
	)

	n, err := NewNonce(ctx, &c, "", false, 5)
	require.NoError(t, err)

	startNonce, _ := n.Next()

	work := func() {
		defer wg.Done()

		for i := 0; i < incrementCount; i++ {
			nonce, err := n.Assign()
			// fail once
			if i == 1 {
				require.NoError(t, n.AddFailedNonce(nonce))
				// fmt.Println(n.failedList)
			}
			require.NoError(t, err)
		}
	}

	for i := 0; i < workerNum; i++ {
		wg.Add(1)
		go work()
	}

	wg.Wait()

	endNonce, _ := n.Next()

	expected := uint64(workerNum*(incrementCount-1)) - startNonce
	require.Equal(t, expected, endNonce)
}
