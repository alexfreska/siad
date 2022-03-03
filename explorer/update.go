package explorer

import (
	"sync"

	"go.sia.tech/core/chain"
	"go.sia.tech/core/consensus"
	"go.sia.tech/core/types"
)

// A Store is a database that stores information about elements, contracts,
// and blocks.
type Store interface {
	AddSiacoinElement(sce types.SiacoinElement) error
	AddSiafundElement(sce types.SiafundElement) error
	AddFileContractElement(fce types.FileContractElement) error
	RemoveElement(id types.ElementID) error
	SiacoinElement(id types.ElementID) (types.SiacoinElement, error)
	SiafundElement(id types.ElementID) (types.SiafundElement, error)
	FileContractElement(id types.ElementID) (types.FileContractElement, error)
	ChainStats(index types.ChainIndex) (ChainStats, error)
	AddChainStats(index types.ChainIndex, stats ChainStats) error
	BlockIndex() (types.ChainIndex, error)
	SetBlockIndex(index types.ChainIndex) error
	SiacoinBalance(address types.Address) (types.Currency, error)
	SiacoinBalanceAdjust(address types.Address, amount types.Currency, add bool) error
	SiafundBalance(address types.Address) (uint64, error)
	SiafundBalanceAdjust(address types.Address, amount uint64, add bool) error
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
func (e *Explorer) ProcessChainApplyUpdate(cau *chain.ApplyUpdate, _ bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

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

	for _, elem := range cau.SpentSiacoins {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
		if err := e.db.SiacoinBalanceAdjust(elem.Address, elem.Value, false); err != nil {
			return err
		}
		stats.SpentSiacoinsCount++
	}
	for _, elem := range cau.SpentSiafunds {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
		if err := e.db.SiafundBalanceAdjust(elem.Address, elem.Value, false); err != nil {
			return err
		}
		stats.SpentSiafundsCount++
	}
	for _, elem := range cau.ResolvedFileContracts {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
		stats.ResolvedFileContractsCount++
		stats.ActiveContractCount--
		payout := elem.FileContract.MissedHostValue.Add(elem.FileContract.TotalCollateral)
		stats.ActiveContractCost = stats.ActiveContractCost.Sub(payout)
		stats.ActiveContractSize -= elem.FileContract.Filesize
	}

	for _, elem := range cau.NewSiacoinElements {
		if err := e.db.AddSiacoinElement(elem); err != nil {
			return err
		}
		if err := e.db.SiacoinBalanceAdjust(elem.Address, elem.Value, true); err != nil {
			return err
		}
		stats.NewSiacoinsCount++
	}
	for _, elem := range cau.NewSiafundElements {
		if err := e.db.AddSiafundElement(elem); err != nil {
			return err
		}
		if err := e.db.SiafundBalanceAdjust(elem.Address, elem.Value, true); err != nil {
			return err
		}
		stats.NewSiafundsCount++
	}
	for _, elem := range cau.RevisedFileContracts {
		if err := e.db.AddFileContractElement(elem); err != nil {
			return err
		}
		stats.RevisedFileContractsCount++
		stats.TotalContractSize += elem.FileContract.Filesize
		stats.TotalRevisionVolume += elem.FileContract.Filesize
	}
	for _, elem := range cau.NewFileContracts {
		if err := e.db.AddFileContractElement(elem); err != nil {
			return err
		}
		stats.NewFileContractsCount++
		payout := elem.FileContract.MissedHostValue.Add(elem.FileContract.TotalCollateral)
		stats.ActiveContractCost = stats.ActiveContractCost.Add(payout)
		stats.ActiveContractSize += elem.FileContract.Filesize
		stats.TotalContractCost = stats.TotalContractCost.Add(payout)
		stats.TotalContractSize += elem.FileContract.Filesize
	}

	if err := e.db.AddChainStats(stats.ValidationContext.Index, stats); err != nil {
		return err
	}
	if err := e.db.SetBlockIndex(stats.ValidationContext.Index); err != nil {
		return err
	}

	e.vc, e.tipStats = cau.Context, stats
	return nil
}

// ProcessChainRevertUpdate implements chain.Subscriber.
func (e *Explorer) ProcessChainRevertUpdate(cru *chain.RevertUpdate) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, elem := range cru.SpentSiacoins {
		if err := e.db.AddSiacoinElement(elem); err != nil {
			return err
		}
		if err := e.db.SiacoinBalanceAdjust(elem.Address, elem.Value, true); err != nil {
			return err
		}
	}
	for _, elem := range cru.SpentSiafunds {
		if err := e.db.AddSiafundElement(elem); err != nil {
			return err
		}
		if err := e.db.SiafundBalanceAdjust(elem.Address, elem.Value, true); err != nil {
			return err
		}
	}
	for _, elem := range cru.ResolvedFileContracts {
		if err := e.db.AddFileContractElement(elem); err != nil {
			return err
		}
	}

	for _, elem := range cru.NewSiacoinElements {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
		if err := e.db.SiacoinBalanceAdjust(elem.Address, elem.Value, false); err != nil {
			return err
		}
	}
	for _, elem := range cru.NewSiafundElements {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
		if err := e.db.SiafundBalanceAdjust(elem.Address, elem.Value, false); err != nil {
			return err
		}
	}
	for _, elem := range cru.RevisedFileContracts {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
	}
	for _, txn := range cru.Block.Transactions {
		for _, rev := range txn.FileContractRevisions {
			if err := e.db.AddFileContractElement(rev.Parent); err != nil {
				return err
			}
		}
	}
	for _, elem := range cru.NewFileContracts {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
	}

	if err := e.db.SetBlockIndex(cru.Context.Index); err != nil {
		return err
	}
	oldStats, err := e.db.ChainStats(cru.Context.Index)
	if err != nil {
		return err
	}

	// update validation context
	e.vc, e.tipStats = cru.Context, oldStats
	return nil
}

// New creates a new explorer.
func New(vc consensus.ValidationContext, store Store) *Explorer {
	return &Explorer{
		vc: vc,
		db: store,
	}
}
