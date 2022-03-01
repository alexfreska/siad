package explored

import (
	"time"

	"go.sia.tech/core/types"
	"go.sia.tech/siad/v2/explorer"
)

// TxpoolBroadcastRequest is the request for the /txpool/broadcast endpoint.
// It contains the transaction to broadcast and the transactions that it
// depends on.
type TxpoolBroadcastRequest struct {
	DependsOn   []types.Transaction `json:"dependsOn"`
	Transaction types.Transaction   `json:"transaction"`
}

// A SyncerPeerResponse is a unique peer that is being used by the syncer.
type SyncerPeerResponse struct {
	NetAddress string `json:"netAddress"`
}

// A SyncerConnectRequest requests that the syncer connect to a peer.
type SyncerConnectRequest struct {
	NetAddress string `json:"netAddress"`
}

// ConsensusTipResponse contains information about the current consensus state.
type ConsensusTipResponse struct {
	Index types.ChainIndex

	TotalWork  types.Work
	Difficulty types.Work
	OakWork    types.Work
	OakTime    time.Duration

	SiafundPool       types.Currency
	FoundationAddress types.Address
}

// An SiacoinElementResponse contains a Siacoin element
type SiacoinElementResponse types.SiacoinElement

// An SiafundElementResponse contains a Siafund element
type SiafundElementResponse types.SiafundElement

// An FileContractElementResponse contains a Siafund element
type FileContractElementResponse types.FileContractElement

// A ChainStatsResponse contains stats about a block.
type ChainStatsResponse explorer.ChainStats
