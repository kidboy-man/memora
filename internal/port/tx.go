package port

import "context"

type Tx interface {
	IsTx()
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(context.Context, Tx) error) error
}
