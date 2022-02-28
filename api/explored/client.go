package explored

import (
	"go.sia.tech/core/types"
	"go.sia.tech/siad/v2/api"
)

// A Client provides methods for interacting with a explored API server.
type Client struct {
	c api.Client
}

// TxpoolBroadcast broadcasts a transaction to the network.
func (c *Client) TxpoolBroadcast(txn types.Transaction, dependsOn []types.Transaction) (err error) {
	err = c.c.Post("/api/txpool/broadcast", TxpoolBroadcastRequest{dependsOn, txn}, nil)
	return
}

// TxpoolTransactions returns all transactions in the transaction pool.
func (c *Client) TxpoolTransactions() (resp []types.Transaction, err error) {
	err = c.c.Get("/api/txpool/transactions", &resp)
	return
}

// SyncerPeers returns the current peers of the syncer.
func (c *Client) SyncerPeers() (resp []SyncerPeerResponse, err error) {
	err = c.c.Get("/api/syncer/peers", &resp)
	return
}

// SyncerConnect adds the address as a peer of the syncer.
func (c *Client) SyncerConnect(addr string) (err error) {
	err = c.c.Post("/api/syncer/connect", addr, nil)
	return
}

// ConsensusTip reports information about the current consensus state.
func (c *Client) ConsensusTip() (resp ConsensusTipResponse, err error) {
	err = c.c.Get("/api/consensus/tip", &resp)
	return
}

// SiacoinElement gets the Siacoin element with the given ID.
func (c *Client) SiacoinElement(id types.ElementID) (resp SiacoinElementResponse, err error) {
	err = c.c.Post("/api/explorer/element/siacoin", ElementRequest{id}, &resp)
	return
}

// SiafundElement gets the Siafund element with the given ID.
func (c *Client) SiafundElement(id types.ElementID) (resp SiafundElementResponse, err error) {
	err = c.c.Post("/api/explorer/element/siafund", ElementRequest{id}, &resp)
	return
}

// FileContractElement gets the file contract element with the given ID.
func (c *Client) FileContractElement(id types.ElementID) (resp FileContractElementResponse, err error) {
	err = c.c.Post("/api/explorer/element/contract", ElementRequest{id}, &resp)
	return
}

// BlockFacts gets facts about the block at the given height.
func (c *Client) BlockFacts(height uint64) (resp BlockFactsResponse, err error) {
	err = c.c.Post("/api/explorer/block/facts", BlockFactsRequest{height}, &resp)
	return
}

// BlockFactsLatest gets facts about the latest block.
func (c *Client) BlockFactsLatest() (resp BlockFactsResponse, err error) {
	err = c.c.Get("/api/explorer/block/facts/latest", &resp)
	return
}

// NewClient returns a client that communicates with a explored server listening on
// the specified address.
func NewClient(addr, password string) *Client {
	return &Client{
		c: api.Client{
			BaseURL:      addr,
			AuthPassword: password,
		},
	}
}
