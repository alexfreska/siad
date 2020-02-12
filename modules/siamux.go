package modules

import (
	"os"
	"path/filepath"

	"gitlab.com/NebulousLabs/Sia/crypto"
	"gitlab.com/NebulousLabs/Sia/persist"
	"gitlab.com/NebulousLabs/Sia/types"
	"gitlab.com/NebulousLabs/siamux"
	"gitlab.com/NebulousLabs/siamux/mux"
)

const (
	// logfile is the filename of the siamux log file
	logfile = "siamux.log"

	// settingsfile is the filename of the host's persistence file
	settingsFile = "host.json"
)

type (
	// hostKeys represents the host's key pair, it is used to extract only the
	// keys from a host's persistence object
	hostKeys struct {
		PublicKey types.SiaPublicKey `json:"publickey"`
		SecretKey crypto.SecretKey   `json:"secretkey"`
	}

	// siaMuxKeys represents a SiaMux key pair
	siaMuxKeys struct {
		pubKey  mux.ED25519PublicKey
		privKey mux.ED25519SecretKey
	}
)

// NewSiaMux returns a new SiaMux object
func NewSiaMux(persistDir, address string) (*siamux.SiaMux, error) {
	// ensure the persist directory exists
	err := os.MkdirAll(persistDir, 0700)
	if err != nil {
		return nil, err
	}

	// create a logger
	file, err := os.OpenFile(filepath.Join(persistDir, logfile), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	logger := persist.NewLogger(file)

	// create a siamux, if the host's persistence file is at v120 we want to
	// recycle the host's key pair to use in the siamux
	pubKey, privKey, compat := compatLoadKeysFromHostV120(persistDir)
	if compat {
		return siamux.CompatV1421NewWithKeyPair(address, logger, persistDir, privKey, pubKey)
	}
	return siamux.New(address, logger, persistDir)
}

// compatLoadKeysFromHostV120 returns the host's persisted keypair. It only
// does this in case the host's persistence version is 1.2.0, otherwise it
// returns nil.
func compatLoadKeysFromHostV120(persistDir string) (pubKey mux.ED25519PublicKey, privKey mux.ED25519SecretKey, compat bool) {
	persistPath := filepath.Join(persistDir, HostDir, settingsFile)

	// Check if we can load the host's persistence object with metadata header
	// v120, if so we are upgrading from 1.2.0 -> 1.4.3 which means we want to
	// recycle the host's key pair to use in the SiaMux.
	var hk hostKeys
	err := persist.LoadJSON(Hostv120PersistMetadata, &hk, persistPath)
	if err == nil {
		copy(pubKey[:], hk.PublicKey.Key[:])
		copy(privKey[:], hk.SecretKey[:])
		compat = true
		return
	}

	compat = false
	return
}
