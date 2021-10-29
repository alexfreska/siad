package wallet

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"go.sia.tech/core/consensus"
	"go.sia.tech/core/types"
)

// A Seed generates ed25519 keys deterministically from some initial entropy.
type Seed struct {
	entropy [16]byte
}

// String implements fmt.Stringer.
func (s Seed) String() string { return hex.EncodeToString(s.entropy[:]) }

// deriveKeyPair derives the keypair for the specified index.
func (s Seed) deriveKeyPair(index uint64) (keypair [64]byte) {
	buf := make([]byte, len(s.entropy)+8)
	n := copy(buf, s.entropy[:])
	binary.LittleEndian.PutUint64(buf[n:], index)
	seed := types.HashBytes(buf)
	copy(keypair[:], ed25519.NewKeyFromSeed(seed[:]))
	return
}

// PublicKey derives the types.SiaPublicKey for the specified index.
func (s Seed) PublicKey(index uint64) (pk types.PublicKey) {
	key := s.deriveKeyPair(index)
	copy(pk[:], key[32:])
	return pk
}

// PrivateKey derives the ed25519 private key for the specified index.
func (s Seed) PrivateKey(index uint64) ed25519.PrivateKey {
	key := s.deriveKeyPair(index)
	return key[:]
}

// SeedFromEntropy returns the Seed derived from the supplied entropy.
func SeedFromEntropy(entropy [16]byte) Seed {
	return Seed{entropy: entropy}
}

// SeedFromString returns the Seed derived from the supplied string.
func SeedFromString(s string) (Seed, error) {
	var entropy [16]byte
	if n, err := hex.Decode(entropy[:], []byte(s)); err != nil {
		return Seed{}, fmt.Errorf("seed string contained invalid characters: %w", err)
	} else if n != 16 {
		return Seed{}, errors.New("invalid seed string length")
	}
	return SeedFromEntropy(entropy), nil
}

// NewSeed returns a random Seed.
func NewSeed() Seed {
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		panic("insufficient system entropy")
	}
	return SeedFromEntropy(entropy)
}

// A Store stores wallet state.
type Store interface {
	SeedIndex() uint64
	AddAddress(addr types.Address, index uint64) error
	AddressIndex(addr types.Address) (uint64, bool)
	SpendableSiacoinElements() []types.SiacoinElement
	Transactions() []Transaction
}

// A HotWallet tracks spendable outputs controlled by in-memory keys. It can
// generate new addresses and sign transactions.
type HotWallet struct {
	mu    sync.Mutex
	seed  Seed
	store Store
	used  map[types.ElementID]bool
}

// Balance returns the total amount of spendable currency controlled by the
// wallet.
func (w *HotWallet) Balance() types.Currency {
	var sum types.Currency
	for _, o := range w.store.SpendableSiacoinElements() {
		sum = sum.Add(o.Value)
	}
	return sum
}

// NextAddress returns an address controlled by the wallet.
func (w *HotWallet) NextAddress() types.Address {
	w.mu.Lock()
	defer w.mu.Unlock()
	index := w.store.SeedIndex()
	addr := types.StandardAddress(w.seed.PublicKey(index))
	w.store.AddAddress(addr, index)
	return addr
}

// FundTransaction adds inputs worth at least amount to txn. It returns the IDs
// of the added inputs, as well as a "discard" function that, when called,
// releases the inputs for use in other transactions.
func (w *HotWallet) FundTransaction(txn *types.Transaction, amount types.Currency, pool []types.Transaction) ([]types.ElementID, func(), error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if amount.IsZero() {
		return nil, func() {}, nil
	}

	// avoid reusing any inputs currently in the transaction pool
	inPool := make(map[types.ElementID]bool)
	for _, ptxn := range pool {
		for _, in := range ptxn.SiacoinInputs {
			inPool[in.Parent.ID] = true
		}
	}

	var outputSum types.Currency
	var fundingElements []types.SiacoinElement
	for _, sce := range w.store.SpendableSiacoinElements() {
		if w.used[sce.ID] || inPool[sce.ID] {
			continue
		}
		fundingElements = append(fundingElements, sce)
		if outputSum = outputSum.Add(sce.Value); outputSum.Cmp(amount) >= 0 {
			break
		}
	}
	if outputSum.Cmp(amount) < 0 {
		return nil, nil, errors.New("insufficient balance")
	} else if outputSum.Cmp(amount) > 0 {
		index := w.store.SeedIndex()
		addr := types.StandardAddress(w.seed.PublicKey(index))
		w.store.AddAddress(addr, index)
		txn.SiacoinOutputs = append(txn.SiacoinOutputs, types.SiacoinOutput{
			Value:   outputSum.Sub(amount),
			Address: addr,
		})
	}

	var toSign []types.ElementID
	for _, sce := range fundingElements {
		index, _ := w.store.AddressIndex(sce.Address)
		txn.SiacoinInputs = append(txn.SiacoinInputs, types.SiacoinInput{
			Parent:      sce,
			SpendPolicy: types.PolicyPublicKey(w.seed.PublicKey(index)),
		})
		toSign = append(toSign, sce.ID)
	}

	for _, o := range fundingElements {
		w.used[o.ID] = true
	}
	discard := func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		for _, o := range fundingElements {
			delete(w.used, o.ID)
		}
	}

	return toSign, discard, nil
}

// SignableInputs returns the inputs of txn that the wallet can sign.
func (w *HotWallet) SignableInputs(txn types.Transaction) []types.ElementID {
	w.mu.Lock()
	defer w.mu.Unlock()
	var ids []types.ElementID
	for _, in := range txn.SiacoinInputs {
		if _, ok := w.store.AddressIndex(in.Parent.Address); ok {
			ids = append(ids, in.Parent.ID)
		}
	}
	return ids
}

// SignTransaction adds signatures to each of the specified inputs.
func (w *HotWallet) SignTransaction(vc consensus.ValidationContext, txn *types.Transaction, toSign []types.ElementID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	inputWithID := func(id types.ElementID) *types.SiacoinInput {
		for i := range txn.SiacoinInputs {
			if in := &txn.SiacoinInputs[i]; in.Parent.ID == id {
				return in
			}
		}
		return nil
	}
	sigHash := vc.SigHash(*txn)
	for _, id := range toSign {
		in := inputWithID(id)
		if in == nil {
			return errors.New("no input with specified ID")
		}
		index, ok := w.store.AddressIndex(in.Parent.Address)
		if !ok {
			return errors.New("no key for specified input")
		}
		in.Signatures = append(in.Signatures, types.InputSignature(types.SignHash(w.seed.PrivateKey(index), sigHash)))
	}
	return nil
}

// A Transaction is a transaction relevant to the wallet, paired with useful
// metadata.
type Transaction struct {
	Raw     types.Transaction
	Index   types.ChainIndex
	ID      types.TransactionID
	Inflow  types.Currency
	Outflow types.Currency
}

// Transactions returns all transactions relevant to the wallet, ordered
// oldest-to-newest.
func (w *HotWallet) Transactions() []Transaction {
	return w.store.Transactions()
}

// NewHotWallet returns a hot wallet using the provided Store and seed.
func NewHotWallet(store Store, seed Seed) *HotWallet {
	return &HotWallet{
		seed:  seed,
		store: store,
		used:  make(map[types.ElementID]bool),
	}
}
