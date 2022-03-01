package explorer

import (
	"go.sia.tech/core/consensus"
	"go.sia.tech/core/types"
)

// ChainStats contains a bunch of statistics about the consensus set as they
// were at a specific block.
type ChainStats struct {
	Block             types.Block
	ValidationContext consensus.ValidationContext

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

// ChainStatsLatest returns stats about the latest black.
func (e *Explorer) ChainStatsLatest() (ChainStats, error) {
	index, err := e.db.BlockIndex()
	if err != nil {
		return ChainStats{}, err
	}
	return e.ChainStats(index)
}

// ChainStats returns stats about the black at the the specified height.
func (e *Explorer) ChainStats(index types.ChainIndex) (ChainStats, error) {
	stats, err := e.db.ChainStats(index)
	if err != nil {
		return ChainStats{}, err
	}
	return stats, nil
}

// SiacoinElement returns the siacoin element associated with the specified ID.
func (e *Explorer) SiacoinElement(id types.ElementID) (types.SiacoinElement, error) {
	sce, err := e.db.SiacoinElement(id)
	if err != nil {
		return types.SiacoinElement{}, err
	}
	return sce, nil
}

// SiafundElement returns the siafund element associated with the specified ID.
func (e *Explorer) SiafundElement(id types.ElementID) (types.SiafundElement, error) {
	sfe, err := e.db.SiafundElement(id)
	if err != nil {
		return types.SiafundElement{}, err
	}
	return sfe, nil
}

// FileContractElement returns the file contract element associated with the specified ID.
func (e *Explorer) FileContractElement(id types.ElementID) (types.FileContractElement, error) {
	fce, err := e.db.FileContractElement(id)
	if err != nil {
		return types.FileContractElement{}, err
	}
	return fce, nil
}
