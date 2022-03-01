package explored

import (
	"encoding/json"
	"fmt"

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
	data, err := json.Marshal(id)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/element/siacoin?id=%s", string(data)), &resp)
	return
}

// SiafundElement gets the Siafund element with the given ID.
func (c *Client) SiafundElement(id types.ElementID) (resp SiafundElementResponse, err error) {
	data, err := json.Marshal(id)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/element/siafund?id=%s", string(data)), &resp)
	return
}

// FileContractElement gets the file contract element with the given ID.
func (c *Client) FileContractElement(id types.ElementID) (resp FileContractElementResponse, err error) {
	data, err := json.Marshal(id)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/element/contract?id=%s", string(data)), &resp)
	return
}

// ChainStats gets stats about the block at the given index.
func (c *Client) ChainStats(index types.ChainIndex) (resp ChainStatsResponse, err error) {
	data, err := json.Marshal(index)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/chain/stats?index=%s", string(data)), &resp)
	return
}

// ChainStatsLatest gets stats about the latest block.
func (c *Client) ChainStatsLatest() (resp ChainStatsResponse, err error) {
	err = c.c.Get("/api/explorer/chain/stats/latest", &resp)
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
