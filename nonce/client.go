package nonce

import (
	"context"
)

type Client interface {
	Nonce(ctx context.Context, privKey string) (nonce uint64, err error)
}
