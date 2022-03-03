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
	return e.db.ChainStats(index)
}

// SiacoinBalance returns the siacoin balance of an address.
func (e *Explorer) SiacoinBalance(address types.Address) (types.Currency, error) {
	return e.db.SiacoinBalance(address)
}

// SiafundBalance returns the siafund balance of an address.
func (e *Explorer) SiafundBalance(address types.Address) (uint64, error) {
	return e.db.SiafundBalance(address)
}

// SiacoinElement returns the siacoin element associated with the specified ID.
func (e *Explorer) SiacoinElement(id types.ElementID) (types.SiacoinElement, error) {
	return e.db.SiacoinElement(id)
}

// SiafundElement returns the siafund element associated with the specified ID.
func (e *Explorer) SiafundElement(id types.ElementID) (types.SiafundElement, error) {
	return e.db.SiafundElement(id)
}

// FileContractElement returns the file contract element associated with the specified ID.
func (e *Explorer) FileContractElement(id types.ElementID) (types.FileContractElement, error) {
	return e.db.FileContractElement(id)
}

// Transaction returns the transaction with the given ID.
func (e *Explorer) Transaction(id types.TransactionID) (types.Transaction, error) {
	return e.db.Transaction(id)
}
