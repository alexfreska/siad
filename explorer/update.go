package explorer

import (
	"sync"

	"go.sia.tech/core/chain"
	"go.sia.tech/core/consensus"
	"go.sia.tech/core/types"
)

type Transaction interface {
	AddSiacoinElement(sce types.SiacoinElement) error
	AddSiafundElement(sfe types.SiafundElement) error
	AddFileContractElement(fce types.FileContractElement) error
	RemoveElement(id types.ElementID) error
	AddChainStats(index types.ChainIndex, stats ChainStats) error
	AddUnspentSiacoinElement(address types.Address, id types.ElementID) error
	AddUnspentSiafundElement(address types.Address, id types.ElementID) error
	RemoveUnspentSiacoinElement(address types.Address, id types.ElementID) error
	RemoveUnspentSiafundElement(address types.Address, id types.ElementID) error
	AddTransaction(txn types.Transaction, addresses []types.Address, block types.ChainIndex) error
	Commit() error
	Rollback() error
}

// A Store is a database that stores information about elements, contracts,
// and blocks.
type Store interface {
	ChainStats(index types.ChainIndex) (ChainStats, error)
	SiacoinElement(id types.ElementID) (types.SiacoinElement, error)
	SiafundElement(id types.ElementID) (types.SiafundElement, error)
	FileContractElement(id types.ElementID) (types.FileContractElement, error)
	UnspentSiacoinElements(address types.Address) ([]types.ElementID, error)
	UnspentSiafundElements(address types.Address) ([]types.ElementID, error)
	Transaction(id types.TransactionID) (types.Transaction, error)
	Transactions(address types.Address, amount int) ([]types.TransactionID, error)
	CreateTransaction() (Transaction, error)
}

// An Explorer contains a database storing information about blocks, outputs,
// contracts.
type Explorer struct {
	db       Store
	mu       sync.Mutex
	tipStats ChainStats
	vc       consensus.ValidationContext
}

// ProcessChainApplyUpdate implements chain.Subscriber.
func (e *Explorer) ProcessChainApplyUpdate(cau *chain.ApplyUpdate, mayCommit bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.CreateTransaction()
	if err != nil {
		return err
	}
	// Will be ignored if tx is committed later
	defer tx.Rollback()

	stats := ChainStats{
		Block:               cau.Block,
		ValidationContext:   cau.Context,
		ActiveContractCost:  e.tipStats.ActiveContractCost,
		ActiveContractCount: e.tipStats.ActiveContractCount,
		ActiveContractSize:  e.tipStats.ActiveContractSize,
		TotalContractCost:   e.tipStats.TotalContractCost,
		TotalContractSize:   e.tipStats.TotalContractSize,
		TotalRevisionVolume: e.tipStats.TotalRevisionVolume,
	}

	for _, txn := range cau.Block.Transactions {
		// get a unique list of all addresses involved in transaction
		var addrMap map[types.Address]struct{}
		for _, elem := range txn.SiacoinInputs {
			addrMap[elem.Parent.Address] = struct{}{}
		}
		for _, elem := range txn.SiacoinOutputs {
			addrMap[elem.Address] = struct{}{}
		}
		for _, elem := range txn.SiafundInputs {
			addrMap[elem.Parent.Address] = struct{}{}
		}
		for _, elem := range txn.SiafundOutputs {
			addrMap[elem.Address] = struct{}{}
		}
		var addrs []types.Address
		for addr := range addrMap {
			addrs = append(addrs, addr)
		}
		if err := tx.AddTransaction(txn, addrs, stats.Block.Header.Index()); err != nil {
			return err
		}
	}

	for _, elem := range cau.SpentSiacoins {
		if err := tx.RemoveElement(elem.ID); err != nil {
			return err
		}
		if err := tx.RemoveUnspentSiacoinElement(elem.Address, elem.ID); err != nil {
			return err
		}
		stats.SpentSiacoinsCount++
	}
	for _, elem := range cau.SpentSiafunds {
		if err := tx.RemoveElement(elem.ID); err != nil {
			return err
		}
		if err := tx.RemoveUnspentSiafundElement(elem.Address, elem.ID); err != nil {
			return err
		}
		stats.SpentSiafundsCount++
	}
	for _, elem := range cau.ResolvedFileContracts {
		if err := tx.RemoveElement(elem.ID); err != nil {
			return err
		}
		stats.ResolvedFileContractsCount++
		stats.ActiveContractCount--
		payout := elem.FileContract.MissedHostValue.Add(elem.FileContract.TotalCollateral)
		stats.ActiveContractCost = stats.ActiveContractCost.Sub(payout)
		stats.ActiveContractSize -= elem.FileContract.Filesize
	}

	for _, elem := range cau.NewSiacoinElements {
		if err := tx.AddSiacoinElement(elem); err != nil {
			return err
		}
		if err := tx.AddUnspentSiacoinElement(elem.Address, elem.ID); err != nil {
			return err
		}
		stats.NewSiacoinsCount++
	}
	for _, elem := range cau.NewSiafundElements {
		if err := tx.AddSiafundElement(elem); err != nil {
			return err
		}
		if err := tx.AddUnspentSiafundElement(elem.Address, elem.ID); err != nil {
			return err
		}
		stats.NewSiafundsCount++
	}
	for _, elem := range cau.RevisedFileContracts {
		if err := tx.AddFileContractElement(elem); err != nil {
			return err
		}
		stats.RevisedFileContractsCount++
		stats.TotalContractSize += elem.FileContract.Filesize
		stats.TotalRevisionVolume += elem.FileContract.Filesize
	}
	for _, elem := range cau.NewFileContracts {
		if err := tx.AddFileContractElement(elem); err != nil {
			return err
		}
		stats.NewFileContractsCount++
		payout := elem.FileContract.MissedHostValue.Add(elem.FileContract.TotalCollateral)
		stats.ActiveContractCost = stats.ActiveContractCost.Add(payout)
		stats.ActiveContractSize += elem.FileContract.Filesize
		stats.TotalContractCost = stats.TotalContractCost.Add(payout)
		stats.TotalContractSize += elem.FileContract.Filesize
	}

	if err := tx.AddChainStats(stats.ValidationContext.Index, stats); err != nil {
		return err
	}

	if mayCommit {
		if err := tx.Commit(); err != nil {
			return err
		}
		e.vc, e.tipStats = cau.Context, stats
	}
	return nil
}

// ProcessChainRevertUpdate implements chain.Subscriber.
func (e *Explorer) ProcessChainRevertUpdate(cru *chain.RevertUpdate) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.CreateTransaction()
	if err != nil {
		return err
	}

	for _, elem := range cru.SpentSiacoins {
		if err := tx.AddSiacoinElement(elem); err != nil {
			return err
		}
		if err := tx.AddUnspentSiacoinElement(elem.Address, elem.ID); err != nil {
			return err
		}
	}
	for _, elem := range cru.SpentSiafunds {
		if err := tx.AddSiafundElement(elem); err != nil {
			return err
		}
		if err := tx.AddUnspentSiafundElement(elem.Address, elem.ID); err != nil {
			return err
		}
	}
	for _, elem := range cru.ResolvedFileContracts {
		if err := tx.AddFileContractElement(elem); err != nil {
			return err
		}
	}

	for _, elem := range cru.NewSiacoinElements {
		if err := tx.RemoveElement(elem.ID); err != nil {
			return err
		}
		if err := tx.RemoveUnspentSiacoinElement(elem.Address, elem.ID); err != nil {
			return err
		}
	}
	for _, elem := range cru.NewSiafundElements {
		if err := tx.RemoveElement(elem.ID); err != nil {
			return err
		}
		if err := tx.RemoveUnspentSiafundElement(elem.Address, elem.ID); err != nil {
			return err
		}
	}
	for _, elem := range cru.RevisedFileContracts {
		if err := tx.RemoveElement(elem.ID); err != nil {
			return err
		}
	}
	for _, txn := range cru.Block.Transactions {
		for _, rev := range txn.FileContractRevisions {
			if err := tx.AddFileContractElement(rev.Parent); err != nil {
				return err
			}
		}
	}
	for _, elem := range cru.NewFileContracts {
		if err := tx.RemoveElement(elem.ID); err != nil {
			return err
		}
	}

	oldStats, err := e.ChainStats(cru.Context.Index)
	if err != nil {
		return err
	}

	// update validation context
	e.vc, e.tipStats = cru.Context, oldStats
	return tx.Commit()
}

// New creates a new explorer.
func New(vc consensus.ValidationContext, store Store) *Explorer {
	return &Explorer{
		vc: vc,
		db: store,
	}
}
