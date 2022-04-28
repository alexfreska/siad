package miner

import (
	"errors"
	"log"
	"sync"

	"go.sia.tech/core/chain"
	"go.sia.tech/core/types"
)

type (
	// A Miner mines blocks.
	Miner interface {
		MineBlock() types.Block
		Address(addr types.Address) error
	}

	// A ChainManager manages blockchain state.
	ChainManager interface {
		AddTipBlock(b types.Block) error
	}

	// A Syncer can connect to other peers and synchronize the blockchain.
	Syncer interface {
		BroadcastBlock(block types.Block)
	}
)

// MiningManager manages an internal block miner.
type MiningManager struct {
	mu   sync.Mutex
	done chan struct{}
	c    ChainManager
	m    Miner
	s    Syncer
}

func (m *MiningManager) run() {
	for {
		select {
		case <-m.done:
			return
		default:
		}
		b := m.m.MineBlock()

		// give it to ourselves
		if err := m.c.AddTipBlock(b); err != nil {
			if !errors.Is(err, chain.ErrUnknownIndex) {
				log.Println("Couldn't add block:", err)
			}
			continue
		}
		log.Println("mined block", b.Index())

		// broadcast it
		m.s.BroadcastBlock(b)
	}
}

// Stop tries to stop the running miner.
func (m *MiningManager) Stop() (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.done == nil {
		log.Println("Miner already stopped")
		return
	}
	m.done <- struct{}{}
	close(m.done)
	m.done = nil
	log.Println("Stopped miner")
	return
}

// Start tries to start the miner.
func (m *MiningManager) Start() (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.done != nil {
		log.Println("Miner already running")
		return
	}
	m.done = make(chan struct{})
	go m.run()
	log.Println("Started miner")
	return
}

// Address sets the miners reward address
func (m *MiningManager) Address(addr types.Address) (err error) {
	return m.m.Address(addr)
}

// New returns a MiningManager initialized with the provided state.
func New(c ChainManager, m Miner, s Syncer) *MiningManager {
	return &MiningManager{
		done: nil,
		c:    c,
		m:    m,
		s:    s,
	}
}
