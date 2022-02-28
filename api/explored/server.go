package explored

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"go.sia.tech/core/consensus"
	"go.sia.tech/core/types"
	"go.sia.tech/siad/v2/api"
	"go.sia.tech/siad/v2/explorer"
)

type (
	// A Syncer can connect to other peers and synchronize the blockchain.
	Syncer interface {
		Addr() string
		Peers() []string
		Connect(addr string) error
		BroadcastTransaction(txn types.Transaction, dependsOn []types.Transaction)
	}

	// A TransactionPool can validate and relay unconfirmed transactions.
	TransactionPool interface {
		Transactions() []types.Transaction
		AddTransaction(txn types.Transaction) error
	}

	// A ChainManager manages blockchain state.
	ChainManager interface {
		TipContext() consensus.ValidationContext
	}

	// An Explorer contains a database storing information about blocks, outputs,
	// contracts.
	Explorer interface {
		SiacoinElement(id types.ElementID) (types.SiacoinElement, bool)
		SiafundElement(id types.ElementID) (types.SiafundElement, bool)
		FileContractElement(id types.ElementID) (types.FileContractElement, bool)
		BlockFacts(height uint64) (explorer.BlockFacts, bool)
		LatestBlockFacts() (explorer.BlockFacts, bool)
	}
)

type server struct {
	s  Syncer
	e  Explorer
	cm ChainManager
	tp TransactionPool
}

func (s *server) txpoolBroadcastHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var tbr TxpoolBroadcastRequest
	if err := json.NewDecoder(req.Body).Decode(&tbr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, txn := range tbr.DependsOn {
		if err := s.tp.AddTransaction(txn); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if err := s.tp.AddTransaction(tbr.Transaction); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.s.BroadcastTransaction(tbr.Transaction, tbr.DependsOn)
}

func (s *server) txpoolTransactionsHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	api.WriteJSON(w, s.tp.Transactions())
}

func (s *server) syncerPeersHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	ps := s.s.Peers()
	sps := make([]SyncerPeerResponse, len(ps))
	for i, peer := range ps {
		sps[i] = SyncerPeerResponse{
			NetAddress: peer,
		}
	}
	api.WriteJSON(w, sps)
}

func (s *server) syncerConnectHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var scr SyncerConnectRequest
	if err := json.NewDecoder(req.Body).Decode(&scr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.s.Connect(scr.NetAddress); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (s *server) consensusTipHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	vc := s.cm.TipContext()
	api.WriteJSON(w, ConsensusTipResponse{
		Index:             vc.Index,
		TotalWork:         vc.TotalWork,
		Difficulty:        vc.Difficulty,
		OakWork:           vc.OakWork,
		OakTime:           vc.OakTime,
		SiafundPool:       vc.SiafundPool,
		FoundationAddress: vc.FoundationAddress,
	})
}

func (s *server) explorerElementSiacoinHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var er ElementRequest
	if err := json.NewDecoder(req.Body).Decode(&er); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	elem, ok := s.e.SiacoinElement(er.ID)
	if !ok {
		http.Error(w, "no such element", http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, elem)
}

func (s *server) explorerElementSiafundHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var er ElementRequest
	if err := json.NewDecoder(req.Body).Decode(&er); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	elem, ok := s.e.SiafundElement(er.ID)
	if !ok {
		http.Error(w, "no such element", http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, elem)
}

func (s *server) explorerElementContractHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var er ElementRequest
	if err := json.NewDecoder(req.Body).Decode(&er); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	elem, ok := s.e.FileContractElement(er.ID)
	if !ok {
		http.Error(w, "no such element", http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, elem)
}

func (s *server) explorerBlockFactsHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var bfr BlockFactsRequest
	if err := json.NewDecoder(req.Body).Decode(&bfr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	facts, ok := s.e.BlockFacts(bfr.Height)
	if !ok {
		http.Error(w, "no block facts for that height", http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, facts)
}

func (s *server) explorerBlockFactsLatestHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	facts, ok := s.e.LatestBlockFacts()
	if !ok {
		http.Error(w, "no block facts for that height", http.StatusInternalServerError)
		return
	}
	api.WriteJSON(w, facts)
}

// NewServer returns an HTTP handler that serves the explored API.
func NewServer(cm ChainManager, s Syncer, tp TransactionPool, e Explorer) http.Handler {
	srv := server{
		cm: cm,
		s:  s,
		tp: tp,
		e:  e,
	}
	mux := httprouter.New()

	mux.GET("/api/txpool/transactions", srv.txpoolTransactionsHandler)
	mux.POST("/api/txpool/broadcast", srv.txpoolBroadcastHandler)

	mux.GET("/api/syncer/peers", srv.syncerPeersHandler)
	mux.POST("/api/syncer/connect", srv.syncerConnectHandler)

	mux.GET("/api/consensus/tip", srv.consensusTipHandler)

	mux.POST("/api/explorer/element/siacoin", srv.explorerElementSiacoinHandler)
	mux.POST("/api/explorer/element/siafund", srv.explorerElementSiafundHandler)
	mux.POST("/api/explorer/element/contract", srv.explorerElementContractHandler)
	mux.POST("/api/explorer/block/facts", srv.explorerBlockFactsHandler)
	mux.GET("/api/explorer/block/facts/latest", srv.explorerBlockFactsLatestHandler)

	return mux
}
