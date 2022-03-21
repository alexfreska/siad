package explorer_test

import (
	"math"
	"testing"

	"go.sia.tech/core/chain"
	"go.sia.tech/core/types"
	"go.sia.tech/siad/v2/explorer"
	"go.sia.tech/siad/v2/internal/chainutil"
	"go.sia.tech/siad/v2/internal/explorerutil"
	"go.sia.tech/siad/v2/internal/walletutil"
	"go.sia.tech/siad/v2/wallet"
)

func TestSiacoinElements(t *testing.T) {
	sim := chainutil.NewChainSim()
	cm := chain.NewManager(chainutil.NewEphemeralStore(sim.Genesis), sim.Context)

	explorerStore, err := explorerutil.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	e := explorer.NewExplorer(sim.Genesis.Context, explorerStore)
	cm.AddSubscriber(e, cm.Tip())

	walletStore := walletutil.NewEphemeralStore()
	cm.AddSubscriber(walletStore, cm.Tip())
	w := wallet.NewHotWallet(walletStore, wallet.NewSeed())

	// fund the wallet with 100 coins
	ourAddr := w.NextAddress()
	fund := types.SiacoinOutput{Value: types.Siacoins(100), Address: ourAddr}
	if err := cm.AddTipBlock(sim.MineBlockWithSiacoinOutputs(fund)); err != nil {
		t.Fatal(err)
	}

	// wallet should now have a transaction, one element, and a non-zero balance

	// mine 5 blocks, each containing a transaction that sends some coins to
	// the void and some to ourself
	for i := 0; i < 5; i++ {
		sendAmount := types.Siacoins(7)
		txn := types.Transaction{
			SiacoinOutputs: []types.SiacoinOutput{{
				Address: types.VoidAddress,
				Value:   sendAmount,
			}},
		}
		if toSign, _, err := w.FundTransaction(&txn, sendAmount, nil); err != nil {
			t.Fatal(err)
		} else if err := w.SignTransaction(sim.Context, &txn, toSign); err != nil {
			t.Fatal(err)
		}

		if err := cm.AddTipBlock(sim.MineBlockWithTxns(txn)); err != nil {
			t.Fatal(err)
		}

		balance, err := e.SiacoinBalance(w.Address())
		if err != nil {
			t.Fatal(err)
		}
		if !w.Balance().Equals(balance) {
			t.Fatal("balances don't equal")
		}

		outputs, err := e.UnspentSiacoinElements(w.Address())
		if err != nil {
			t.Fatal(err)
		}
		if len(outputs) != 1 {
			t.Fatal("wrong amount of outputs")
		}
		elem, err := e.SiacoinElement(outputs[0])
		if err != nil {
			t.Fatal(err)
		}
		if !w.Balance().Equals(elem.Value) {
			t.Fatal("output value doesn't equal balance")
		}
		txns, err := e.Transactions(w.Address(), math.MaxInt64)
		if err != nil {
			t.Fatal(err)
		}
		if len(txns) != 1 {
			t.Fatal("wrong number of transactions")
		}
		if txn.ID() != txns[0] {
			t.Fatal("wrong transaction")
		}
		txns0, err := e.Transaction(txns[0])
		if err != nil {
			t.Fatal(err)
		}
		if txn.ID() != txns0.ID() {
			t.Fatal("wrong transaction")
		}
	}
}
