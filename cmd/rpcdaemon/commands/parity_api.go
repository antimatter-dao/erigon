package commands

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/core/state"
)

// ParityAPI the interface for the parity_ RPC commands
type ParityAPI interface {
	ListStorageKeys(ctx context.Context, account common.Address, quantity int, offset *hexutil.Bytes) ([]hexutil.Bytes, error)
}

// ParityAPIImpl data structure to store things needed for parity_ commands
type ParityAPIImpl struct {
	db kv.RoDB
}

// NewParityAPIImpl returns ParityAPIImpl instance
func NewParityAPIImpl(db kv.RoDB) *ParityAPIImpl {
	return &ParityAPIImpl{
		db: db,
	}
}

// ListStorageKeys implements parity_listStorageKeys. Returns all storage keys of the given address
func (api *ParityAPIImpl) ListStorageKeys(ctx context.Context, account common.Address, quantity int, offset *hexutil.Bytes) ([]hexutil.Bytes, error) {
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("listStorageKeys cannot open tx: %w", err)
	}
	defer tx.Rollback()
	a, err := state.NewPlainStateReader(tx).ReadAccountData(account)
	if err != nil {
		return nil, err
	} else if a == nil {
		return nil, fmt.Errorf("acc not found")
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, a.GetIncarnation())
	seekBytes := append(account.Bytes(), b...)

	c, err := tx.CursorDupSort(kv.PlainState)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	keys := make([]hexutil.Bytes, 0)
	var v []byte
	var seekVal []byte
	if offset != nil {
		seekVal = *offset
	}

	for v, err = c.SeekBothRange(seekBytes, seekVal); v != nil && len(keys) != quantity && err == nil; _, v, err = c.NextDup() {
		if len(v) > common.HashLength {
			keys = append(keys, v[:common.HashLength])
		} else {
			keys = append(keys, v)
		}
	}
	if err != nil {
		return nil, err
	}
	return keys, nil
}
