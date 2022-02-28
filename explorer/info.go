package explorer

import (
	"time"

	"go.sia.tech/core/types"
)

// BlockFacts returns a bunch of statistics about the consensus set as they
// were at a specific block.
type BlockFacts struct {
	Index             types.ChainIndex
	TotalWork         types.Work
	Difficulty        types.Work
	OakWork           types.Work
	OakTime           time.Duration
	GenesisTimestamp  time.Time
	SiafundPool       types.Currency
	FoundationAddress types.Address

	// Transaction type counts.
	SpentSiacoinsCount         uint64
	SpentSiafundsCount         uint64
	NewSiacoinsCount           uint64
	NewSiafundsCount           uint64
	NewFileContractsCount      uint64
	RevisedFileContractsCount  uint64
	ResolvedFileContractsCount uint64

	// Facts about file contracts.
	ActiveContractCost  types.Currency
	ActiveContractCount uint64
	ActiveContractSize  uint64
	TotalContractCost   types.Currency
	TotalContractSize   uint64
	TotalRevisionVolume uint64
}

// LatestBlockFacts returns facts about the latest black.
func (e *Explorer) LatestBlockFacts() (BlockFacts, bool) {
	height, err := e.db.BlockHeight()
	if err != nil {
		return BlockFacts{}, false
	}
	return e.BlockFacts(height)
}

// BlockFacts returns facts about the black at the the specified height.
func (e *Explorer) BlockFacts(height uint64) (BlockFacts, bool) {
	facts, err := e.db.BlockFacts(height)
	if err != nil {
		return BlockFacts{}, false
	}
	return facts, true
}

// SiacoinElement returns the siacoin element associated with the specified ID.
func (e *Explorer) SiacoinElement(id types.ElementID) (types.SiacoinElement, bool) {
	sce, err := e.db.SiacoinElement(id)
	if err != nil {
		return types.SiacoinElement{}, false
	}
	return sce, true
}

// SiafundElement returns the siafund element associated with the specified ID.
func (e *Explorer) SiafundElement(id types.ElementID) (types.SiafundElement, bool) {
	sfe, err := e.db.SiafundElement(id)
	if err != nil {
		return types.SiafundElement{}, false
	}
	return sfe, true
}

// FileContractElement returns the file contract element associated with the specified ID.
func (e *Explorer) FileContractElement(id types.ElementID) (types.FileContractElement, bool) {
	fce, err := e.db.FileContractElement(id)
	if err != nil {
		return types.FileContractElement{}, false
	}
	return fce, true
}
