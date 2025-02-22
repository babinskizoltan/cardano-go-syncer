package core

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/blinklabs-io/gouroboros/ledger"
)

type BlockPoint struct {
	BlockSlot   uint64 `json:"slot"`
	BlockHash   []byte `json:"hash"`
	BlockNumber uint64 `json:"num"`
}

type BlockHeader struct {
	BlockSlot   uint64 `json:"slot"`
	BlockHash   []byte `json:"hash"`
	BlockNumber uint64 `json:"num"`
	EraID       uint8  `json:"era"`
	EraName     string `json:"-"`
}

type FullBlock struct {
	BlockSlot   uint64 `json:"slot"`
	BlockHash   []byte `json:"hash"`
	BlockNumber uint64 `json:"num"`
	EraID       uint8  `json:"era"`
	EraName     string `json:"-"`
	Txs         []*Tx  `json:"txs"`
}

type Tx struct {
	Hash     string      `json:"hash"`
	Metadata []byte      `json:"metadata"`
	Inputs   []*TxInput  `json:"inputs"`
	Outputs  []*TxOutput `json:"outputs"`
	Fee      uint64      `json:"fee"`
}

type TxInput struct {
	Hash  string `json:"id"`
	Index uint32 `json:"index"`
}

type TxOutput struct {
	Address string `json:"address"`
	Amount  uint64 `json:"amount"`
}

func GetBlockHeaderFromBlockInfo(blockType uint, blockInfo interface{}, nextBlockNumber uint64) (*BlockHeader, error) {
	var blockHeaderFull ledger.BlockHeader

	// /home/bbs/go/pkg/mod/github.com/blinklabs-io/gouroboros@v0.69.3/ledger/block.go
	// func NewBlockHeaderFromCbor(blockType uint, data []byte) (BlockHeader, error) {
	switch blockType {
	case ledger.BlockTypeByronEbb:
		blockHeaderFull = blockInfo.(*ledger.ByronEpochBoundaryBlockHeader)
	case ledger.BlockTypeByronMain:
		blockHeaderFull = blockInfo.(*ledger.ByronMainBlockHeader)
	case ledger.BlockTypeShelley, ledger.BlockTypeAllegra, ledger.BlockTypeMary, ledger.BlockTypeAlonzo:
		blockHeaderFull = blockInfo.(*ledger.ShelleyBlockHeader)
	case ledger.BlockTypeBabbage, ledger.BlockTypeConway:
		blockHeaderFull = blockInfo.(*ledger.BabbageBlockHeader)
	}

	// nolint
	blockHash, _ := hex.DecodeString(blockHeaderFull.Hash())

	blockNumber := blockHeaderFull.BlockNumber()
	if blockNumber == 0 {
		blockNumber = nextBlockNumber
	} else if blockNumber != nextBlockNumber {
		return nil, fmt.Errorf("invalid number of block: expected %d vs %d", nextBlockNumber, blockNumber)
	}

	return &BlockHeader{
		BlockSlot:   blockHeaderFull.SlotNumber(),
		BlockHash:   blockHash,
		BlockNumber: blockNumber,
		EraID:       blockHeaderFull.Era().Id,
		EraName:     blockHeaderFull.Era().Name,
	}, nil
}

func NewFullBlock(bh *BlockHeader, txs []*Tx) *FullBlock {
	return &FullBlock{
		BlockSlot:   bh.BlockSlot,
		BlockHash:   bh.BlockHash,
		BlockNumber: bh.BlockNumber,
		EraID:       bh.EraID,
		EraName:     bh.EraName,
		Txs:         txs,
	}
}

func NewTransaction(ledgerTx ledger.Transaction) *Tx {
	tx := &Tx{
		Hash: ledgerTx.Hash(),
		Fee:  ledgerTx.Fee(),
	}

	if ledgerTx.Metadata() != nil && ledgerTx.Metadata().Cbor() != nil {
		tx.Metadata = ledgerTx.Metadata().Cbor()
	}

	inputs, outputs := ledgerTx.Inputs(), ledgerTx.Outputs()

	if ln := len(inputs); ln > 0 {
		tx.Inputs = make([]*TxInput, ln)
		for j, inp := range inputs {
			tx.Inputs[j] = &TxInput{
				Hash:  inp.Id().String(),
				Index: inp.Index(),
			}
		}
	}

	if ln := len(outputs); ln > 0 {
		tx.Outputs = make([]*TxOutput, ln)
		for j, out := range outputs {
			tx.Outputs[j] = &TxOutput{
				Address: out.Address().String(),
				Amount:  out.Amount(),
			}
		}
	}

	return tx
}

func NewTransactions(ledgerTxs []ledger.Transaction) []*Tx {
	if len(ledgerTxs) == 0 {
		return nil
	}

	result := make([]*Tx, len(ledgerTxs))
	for i, x := range ledgerTxs {
		result[i] = NewTransaction(x)
	}

	return result
}

func (fb FullBlock) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("number = %d, hash = %s, tx count = %d\n", fb.BlockNumber, hex.EncodeToString(fb.BlockHash), len(fb.Txs)))
	for _, tx := range fb.Txs {
		var (
			sbInp strings.Builder
			sbOut strings.Builder
		)

		for _, x := range tx.Inputs {
			if sbInp.Len() > 0 {
				sbInp.WriteString(", ")
			}

			sbInp.WriteString("[")
			sbInp.WriteString(x.Hash)
			sbInp.WriteString(", ")
			sbInp.WriteString(strconv.FormatUint(uint64(x.Index), 10))
			sbInp.WriteString("]")
		}

		for i, x := range tx.Outputs {
			if sbOut.Len() > 0 {
				sbOut.WriteString(", ")
			}

			sbOut.WriteString("[")
			sbOut.WriteString(strconv.Itoa(i))
			sbOut.WriteString(", ")
			sbOut.WriteString(x.Address)
			sbOut.WriteString(", ")
			sbOut.WriteString(strconv.FormatUint(x.Amount, 10))
			sbOut.WriteString("]")
		}

		sb.WriteString(fmt.Sprintf("  tx hash = %s, fee = %d\n", tx.Hash, tx.Fee))
		if tx.Metadata != nil {
			sb.WriteString(fmt.Sprintf("  meta = %s\n", string(tx.Metadata)))
		}

		sb.WriteString(fmt.Sprintf("   inputs = %s\n", sbInp.String()))
		sb.WriteString(fmt.Sprintf("  outputs = %s\n", sbOut.String()))
	}

	return sb.String()
}

func (ti TxInput) Key() []byte {
	return []byte(fmt.Sprintf("%s_%d", ti.Hash, ti.Index))
}

func (fb FullBlock) Key() []byte {
	return EncodeUint64ToBytes(fb.BlockNumber)
}

func EncodeUint64ToBytes(value uint64) []byte {
	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, value)

	return result
}
