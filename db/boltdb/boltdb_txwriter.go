package boltdb

import (
	"encoding/json"
	"fmt"
	"igorcrevar/cardano-go-syncer/core"

	"github.com/boltdb/bolt"
)

type txOperation func(tx *bolt.Tx) error

type BoltDbTransactionWriter struct {
	db         *bolt.DB
	operations []txOperation
}

var _ core.DbTransactionWriter = (*BoltDbTransactionWriter)(nil)

func (tw *BoltDbTransactionWriter) SetLatestBlockPoint(point *core.BlockPoint) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(tx *bolt.Tx) error {
		bytes, err := json.Marshal(point)
		if err != nil {
			return fmt.Errorf("could not marshal latest block point: %v", err)
		}

		if err = tx.Bucket(latestBlockPointBucket).Put(defaultKey, bytes); err != nil {
			return fmt.Errorf("latest block point write error: %v", err)
		}

		return nil
	})

	return tw
}

func (tw *BoltDbTransactionWriter) AddTxOutput(txInput core.TxInput, txOutput *core.TxOutput) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(tx *bolt.Tx) error {
		bytes, err := json.Marshal(txOutput)
		if err != nil {
			return fmt.Errorf("could not marshal tx output: %v", err)
		}

		if err = tx.Bucket(txOutputsBucket).Put(txInput.Key(), bytes); err != nil {
			return fmt.Errorf("tx output write error: %v", err)
		}

		return nil
	})

	return tw
}

func (tw *BoltDbTransactionWriter) AddConfirmedBlock(block *core.FullBlock) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(tx *bolt.Tx) error {
		bytes, err := json.Marshal(block)
		if err != nil {
			return fmt.Errorf("could not marshal confirmed block: %v", err)
		}

		if err = tx.Bucket(unprocessedBlocksBucket).Put(block.Key(), bytes); err != nil {
			return fmt.Errorf("confirmed block write error: %v", err)
		}

		return nil
	})

	return tw
}

func (tw *BoltDbTransactionWriter) RemoveTxOutputs(txInputs []*core.TxInput) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(tx *bolt.Tx) error {
		bucket := tx.Bucket(txOutputsBucket)
		for _, inp := range txInputs {
			if err := bucket.Delete(inp.Key()); err != nil {
				return err
			}
		}

		return nil
	})

	return tw
}

func (tw *BoltDbTransactionWriter) Execute() error {
	defer func() {
		tw.operations = nil
	}()

	return tw.db.Update(func(tx *bolt.Tx) error {
		for _, op := range tw.operations {
			if err := op(tx); err != nil {
				return err
			}
		}

		return nil
	})
}
