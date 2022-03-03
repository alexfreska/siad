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
		SiacoinElement(id types.ElementID) (types.SiacoinElement, error)
		SiafundElement(id types.ElementID) (types.SiafundElement, error)
		FileContractElement(id types.ElementID) (types.FileContractElement, error)
		ChainStats(index types.ChainIndex) (explorer.ChainStats, error)
		ChainStatsLatest() (explorer.ChainStats, error)
		SiacoinBalance(address types.Address) (types.Currency, error)
		SiafundBalance(address types.Address) (uint64, error)
		Transaction(id types.TransactionID) (types.Transaction, error)
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

func (s *server) explorerElementSiacoinHandler(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	var id types.ElementID
	if err := json.Unmarshal([]byte(p.ByName("id")), &id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	elem, err := s.e.SiacoinElement(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, elem)
}

func (s *server) explorerElementSiafundHandler(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	var id types.ElementID
	if err := json.Unmarshal([]byte(p.ByName("id")), &id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	elem, err := s.e.SiafundElement(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, elem)
}

func (s *server) explorerElementContractHandler(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	var id types.ElementID
	if err := json.Unmarshal([]byte(p.ByName("id")), &id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	elem, err := s.e.FileContractElement(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, elem)
}

func (s *server) explorerChainStatsHandler(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	if p.ByName("index") == "latest" {
		facts, err := s.e.ChainStatsLatest()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		api.WriteJSON(w, facts)
		return
	}

	var index types.ChainIndex
	if err := json.Unmarshal([]byte(p.ByName("index")), &index); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	facts, err := s.e.ChainStats(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, facts)
}

func (s *server) explorerSearchHandler(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	var id types.ElementID
	if err := json.Unmarshal([]byte(p.ByName("id")), &id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var response ExplorerSearchResponse
	if elem, err := s.e.SiacoinElement(id); err == nil {
		response.Type = "siacoin"
		response.SiacoinElement = elem
	}
	if elem, err := s.e.SiafundElement(id); err == nil {
		response.Type = "siafund"
		response.SiafundElement = elem
	}
	if elem, err := s.e.FileContractElement(id); err == nil {
		response.Type = "contract"
		response.FileContractElement = elem
	}
	api.WriteJSON(w, response)
}

func (s *server) explorerBalanceSiacoinHandler(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	var address types.Address
	if err := json.Unmarshal([]byte(p.ByName("address")), &address); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	balance, err := s.e.SiacoinBalance(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, balance)
}

func (s *server) explorerBalanceSiafundHandler(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	var address types.Address
	if err := json.Unmarshal([]byte(p.ByName("address")), &address); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	balance, err := s.e.SiafundBalance(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, balance)
}

func (s *server) explorerTransactionHandler(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	var id types.TransactionID
	if err := json.Unmarshal([]byte(p.ByName("id")), &id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	txn, err := s.e.Transaction(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.WriteJSON(w, txn)
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

	mux.GET("/api/explorer/element/siacoin/:id", srv.explorerElementSiacoinHandler)
	mux.GET("/api/explorer/element/siafund/:id", srv.explorerElementSiafundHandler)
	mux.GET("/api/explorer/element/contract/:id", srv.explorerElementContractHandler)
	mux.GET("/api/explorer/chain/stats/:index", srv.explorerChainStatsHandler)
	mux.GET("/api/explorer/search/:id", srv.explorerSearchHandler)
	mux.GET("/api/explorer/balance/siacoin/:address", srv.explorerBalanceSiacoinHandler)
	mux.GET("/api/explorer/balance/siafund/:address", srv.explorerBalanceSiafundHandler)
	mux.GET("/api/explorer/transaction/:id", srv.explorerTransactionHandler)

	return mux
}
