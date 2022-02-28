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
	BlockFacts(height uint64) (BlockFacts, error)
	AddBlockFacts(height uint64, facts BlockFacts) error
	RemoveBlockFacts(height uint64) error
	BlockHeight() (uint64, error)
	SetBlockHeight(height uint64) error
}

// An Explorer contains a database storing information about blocks, outputs,
// contracts.
type Explorer struct {
	db Store
	mu sync.Mutex
	vc consensus.ValidationContext
}

// ProcessChainApplyUpdate implements chain.Subscriber.
func (e *Explorer) ProcessChainApplyUpdate(cau *chain.ApplyUpdate, _ bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	oldFacts, err := e.db.BlockFacts(e.vc.Index.Height)
	if err != nil {
		return err
	}

	facts := BlockFacts{
		Index:               cau.Context.Index,
		TotalWork:           cau.Context.TotalWork,
		Difficulty:          cau.Context.Difficulty,
		OakWork:             cau.Context.OakWork,
		OakTime:             cau.Context.OakTime,
		GenesisTimestamp:    cau.Context.GenesisTimestamp,
		SiafundPool:         cau.Context.SiafundPool,
		FoundationAddress:   cau.Context.FoundationAddress,
		ActiveContractCost:  oldFacts.ActiveContractCost,
		ActiveContractCount: oldFacts.ActiveContractCount,
		ActiveContractSize:  oldFacts.ActiveContractSize,
		TotalContractCost:   oldFacts.TotalContractCost,
		TotalContractSize:   oldFacts.TotalContractSize,
		TotalRevisionVolume: oldFacts.TotalRevisionVolume,
	}

	for _, elem := range cau.SpentSiacoins {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
		facts.SpentSiacoinsCount++
	}
	for _, elem := range cau.SpentSiafunds {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
		facts.SpentSiafundsCount++
	}
	for _, elem := range cau.ResolvedFileContracts {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
		facts.ResolvedFileContractsCount++
		facts.ActiveContractCount--
		payout := elem.FileContract.MissedHostValue.Add(elem.FileContract.TotalCollateral)
		facts.ActiveContractCost = facts.ActiveContractCost.Sub(payout)
		facts.ActiveContractSize -= elem.FileContract.Filesize
	}

	for _, elem := range cau.NewSiacoinElements {
		if err := e.db.AddSiacoinElement(elem); err != nil {
			return err
		}
		facts.NewSiacoinsCount++
	}
	for _, elem := range cau.NewSiafundElements {
		if err := e.db.AddSiafundElement(elem); err != nil {
			return err
		}
		facts.NewSiafundsCount++
	}
	for _, elem := range cau.RevisedFileContracts {
		if err := e.db.AddFileContractElement(elem); err != nil {
			return err
		}
		facts.RevisedFileContractsCount++
		facts.TotalContractSize += elem.FileContract.Filesize
		facts.TotalRevisionVolume += elem.FileContract.Filesize
	}
	for _, elem := range cau.NewFileContracts {
		if err := e.db.AddFileContractElement(elem); err != nil {
			return err
		}
		facts.NewFileContractsCount++
		payout := elem.FileContract.MissedHostValue.Add(elem.FileContract.TotalCollateral)
		facts.ActiveContractCost = facts.ActiveContractCost.Add(payout)
		facts.ActiveContractSize += elem.FileContract.Filesize
		facts.TotalContractCost = facts.TotalContractCost.Add(payout)
		facts.TotalContractSize += elem.FileContract.Filesize
	}

	if err := e.db.AddBlockFacts(facts.Index.Height, facts); err != nil {
		return err
	}
	if err := e.db.SetBlockHeight(facts.Index.Height); err != nil {
		return err
	}

	e.vc = cau.Context
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
	}
	for _, elem := range cru.SpentSiafunds {
		if err := e.db.AddSiafundElement(elem); err != nil {
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
	}
	for _, elem := range cru.NewSiafundElements {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
	}
	for _, elem := range cru.RevisedFileContracts {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
	}
	for _, elem := range cru.NewFileContracts {
		if err := e.db.RemoveElement(elem.ID); err != nil {
			return err
		}
	}

	if err := e.db.RemoveBlockFacts(e.vc.Index.Height); err != nil {
		return err
	}
	if err := e.db.SetBlockHeight(cru.Context.Index.Height); err != nil {
		return err
	}

	// update validation context
	e.vc = cru.Context
	return nil
}

// New creates a new explorer.
func New(vc consensus.ValidationContext, store Store) *Explorer {
	return &Explorer{
		vc: vc,
		db: store,
	}
}
