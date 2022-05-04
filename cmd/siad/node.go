package main

import (
	"fmt"
	"os"
	"path/filepath"

	"go.sia.tech/core/chain"
	"go.sia.tech/core/consensus"
	"go.sia.tech/core/types"
	"go.sia.tech/siad/v2/internal/chainutil"
	"go.sia.tech/siad/v2/internal/cpuminer"
	"go.sia.tech/siad/v2/internal/p2putil"
	"go.sia.tech/siad/v2/internal/walletutil"
	"go.sia.tech/siad/v2/miner"
	"go.sia.tech/siad/v2/p2p"
	"go.sia.tech/siad/v2/txpool"
)

type node struct {
	c  *chain.Manager
	tp *txpool.Pool
	s  *p2p.Syncer
	w  *walletutil.JSONStore
	m  *miner.MiningManager
}

func (n *node) run() error {
	return n.s.Run()
}

func (n *node) Close() error {
	errs := []error{
		n.s.Close(),
		n.c.Close(),
	}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func newNode(addr, dir, minerAddr string, c consensus.Checkpoint) (*node, error) {
	chainDir := filepath.Join(dir, "chain")
	if err := os.MkdirAll(chainDir, 0700); err != nil {
		return nil, err
	}
	chainStore, tip, err := chainutil.NewFlatStore(chainDir, c)
	if err != nil {
		return nil, err
	}

	walletDir := filepath.Join(dir, "wallet")
	if err := os.MkdirAll(walletDir, 0700); err != nil {
		return nil, err
	}
	walletStore, walletTip, err := walletutil.NewJSONStore(walletDir, tip.State.Index)
	if err != nil {
		return nil, err
	}

	cm := chain.NewManager(chainStore, tip.State)
	tp := txpool.New(tip.State)
	cm.AddSubscriber(tp, cm.Tip())
	if err := cm.AddSubscriber(walletStore, walletTip); err != nil {
		return nil, fmt.Errorf("couldn't resubscribe wallet at index %v: %w", walletTip, err)
	}

	p2pDir := filepath.Join(dir, "p2p")
	if err := os.MkdirAll(p2pDir, 0700); err != nil {
		return nil, err
	}
	peerStore, err := p2putil.NewJSONStore(p2pDir)
	if err != nil {
		return nil, err
	}
	s, err := p2p.NewSyncer(addr, genesisBlock.ID(), cm, tp, peerStore)
	if err != nil {
		return nil, err
	}

	minerAddrParsed, err := types.ParseAddress(minerAddr)
	if err != nil {
		return nil, err
	}

	mcpu := cpuminer.New(tip.State, minerAddrParsed, tp)
	cm.AddSubscriber(mcpu, cm.Tip())
	m := miner.New(cm, mcpu, s)

	return &node{
		c:  cm,
		tp: tp,
		s:  s,
		w:  walletStore,
		m:  m,
	}, nil
}
