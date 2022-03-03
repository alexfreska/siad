package explored

import (
	"encoding/json"
	"fmt"

	"go.sia.tech/core/types"
	"go.sia.tech/siad/v2/api"
	"go.sia.tech/siad/v2/explorer"
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

// ExplorerSiacoinElement gets the Siacoin element with the given ID.
func (c *Client) ExplorerSiacoinElement(id types.ElementID) (resp types.SiacoinElement, err error) {
	data, err := json.Marshal(id)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/element/siacoin/%s", string(data)), &resp)
	return
}

// ExplorerSiafundElement gets the Siafund element with the given ID.
func (c *Client) ExplorerSiafundElement(id types.ElementID) (resp types.SiafundElement, err error) {
	data, err := json.Marshal(id)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/element/siafund/%s", string(data)), &resp)
	return
}

// ExplorerFileContractElement gets the file contract element with the given ID.
func (c *Client) ExplorerFileContractElement(id types.ElementID) (resp types.FileContractElement, err error) {
	data, err := json.Marshal(id)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/element/contract/%s", string(data)), &resp)
	return
}

// ExplorerChainStats gets stats about the block at the given index.
func (c *Client) ExplorerChainStats(index types.ChainIndex) (resp explorer.ChainStats, err error) {
	data, err := json.Marshal(index)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/chain/stats/%s", string(data)), &resp)
	return
}

// ExplorerChainStatsLatest gets stats about the latest block.
func (c *Client) ExplorerChainStatsLatest() (resp explorer.ChainStats, err error) {
	err = c.c.Get("/api/explorer/chain/stats/latest", &resp)
	return
}

// ExplorerSearch gets information about a given element.
func (c *Client) ExplorerSearch(id types.ElementID) (resp ExplorerSearchResponse, err error) {
	data, err := json.Marshal(id)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/search/%s", string(data)), &resp)
	return
}

// ExplorerSiacoinBalance gets the siacoin balance of an address.
func (c *Client) ExplorerSiacoinBalance(address types.Address) (resp types.Currency, err error) {
	data, err := json.Marshal(address)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/balance/siacoin/%s", string(data)), &resp)
	return
}

// ExplorerSiafundBalance gets the siafund balance of an address.
func (c *Client) ExplorerSiafundBalance(address types.Address) (resp types.Currency, err error) {
	data, err := json.Marshal(address)
	if err != nil {
		return
	}
	err = c.c.Get(fmt.Sprintf("/api/explorer/balance/siafund/%s", string(data)), &resp)
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
