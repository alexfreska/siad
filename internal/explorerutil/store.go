package explorerutil

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"go.sia.tech/core/types"
	"go.sia.tech/siad/v2/explorer"
	_ "modernc.org/sqlite"
)

type SqliteStore struct {
	*sql.DB
}

type Transaction struct {
	*sql.Tx
}

func NewStore(dir string) (*SqliteStore, error) {
	db, err := sql.Open("sqlite", path.Join(dir, "store.db"))
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
CREATE TABLE elements (
    id VARCHAR(128) PRIMARY KEY,
    type VARCHAR(128),
    data BLOB NOT NULL
);

CREATE TABLE chainstats (
    id VARCHAR(128) PRIMARY KEY,
    data BLOB NOT NULL
);

CREATE TABLE unspentElements (
    id VARCHAR(128) PRIMARY KEY,
    type VARCHAR(128),
	address VARCHAR(128)
);

CREATE TABLE transactions (
    id VARCHAR(128) PRIMARY KEY,
    data BLOB NOT NULL
);

CREATE TABLE addressTransactions (
    id VARCHAR(128),
    address VARCHAR(128)
);
`); err != nil && !strings.Contains(err.Error(), "already exists") {
		panic(err)
	}
	return &SqliteStore{db}, nil
}

func (s *SqliteStore) CreateTransaction() (explorer.Transaction, error) {
	tx, err := s.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx}, nil
}

func (s *SqliteStore) SiacoinElement(id types.ElementID) (sce types.SiacoinElement, err error) {
	row := s.QueryRow(`SELECT data FROM elements WHERE id=? AND type=?`, id.String(), "siacoin")

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	if err = json.Unmarshal(data, &sce); err != nil {
		return
	}
	return
}

func (s *SqliteStore) SiafundElement(id types.ElementID) (sfe types.SiafundElement, err error) {
	row := s.QueryRow(`SELECT data FROM elements WHERE id=? AND type=?`, id.String(), "siafund")

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	if err = json.Unmarshal(data, &sfe); err != nil {
		return
	}
	return
}

func (s *SqliteStore) FileContractElement(id types.ElementID) (fce types.FileContractElement, err error) {
	row := s.QueryRow(`SELECT data FROM elements WHERE id=? AND type=?`, id.String(), "contract")

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	if err = json.Unmarshal(data, &fce); err != nil {
		return
	}
	return
}

func (s *SqliteStore) ChainStats(index types.ChainIndex) (stats explorer.ChainStats, err error) {
	row := s.QueryRow(`SELECT data FROM chainstats WHERE id=?`, index.String())

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	if err = json.Unmarshal(data, &stats); err != nil {
		return
	}
	return
}

func parseElementID(str string) (id types.ElementID, err error) {
	_, err = fmt.Sscanf(str, "h:%v:%v", &id.Source, &id.Index)
	return
}

func (s *SqliteStore) UnspentSiacoinElements(address types.Address) ([]types.ElementID, error) {
	rows, err := s.Query(`SELECT data FROM unspentElements WHERE address=? AND type=?`, address.String(), "siacoin")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []types.ElementID
	for rows.Next() {
		var str string
		if err := rows.Scan(&str); err != nil {
			return nil, err
		}

		id, err := parseElementID(str)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SqliteStore) UnspentSiafundElements(address types.Address) ([]types.ElementID, error) {
	rows, err := s.Query(`SELECT data FROM unspentElements WHERE address=? AND type=?`, address.String(), "siafund")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []types.ElementID
	for rows.Next() {
		var str string
		if err := rows.Scan(&str); err != nil {
			return nil, err
		}

		id, err := parseElementID(str)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SqliteStore) Transaction(id types.TransactionID) (txn types.Transaction, err error) {
	row := s.QueryRow(`SELECT data FROM transactions WHERE id=?`, id.String())

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	if err = json.Unmarshal(data, &txn); err != nil {
		return
	}
	return
}

func (s *SqliteStore) Transactions(address types.Address, amount int) ([]types.TransactionID, error) {
	rows, err := s.Query(`SELECT id FROM addressTransactions WHERE address=? LIMIT 0,?`, address.String(), amount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []types.TransactionID
	for rows.Next() {
		var str string
		if err := rows.Scan(&str); err != nil {
			return nil, err
		}

		var idHex string
		if _, err := fmt.Sscanf(str, "txid:%s", &idHex); err != nil {
			return nil, err
		}
		var id types.TransactionID
		if _, err := hex.Decode(id[:], []byte(idHex)); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (tx *Transaction) AddSiacoinElement(sce types.SiacoinElement) error {
	statement, err := tx.Prepare(`INSERT INTO elements(id, type, data) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := json.Marshal(sce)
	if err != nil {
		return err
	}
	_, err = statement.Exec(sce.ID.String(), "siacoin", data)
	return err
}

func (tx *Transaction) AddSiafundElement(sfe types.SiafundElement) error {
	statement, err := tx.Prepare(`INSERT INTO elements(id, type, data) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := json.Marshal(sfe)
	if err != nil {
		return err
	}
	_, err = statement.Exec(sfe.ID.String(), "siafund", data)
	return err
}

func (tx *Transaction) AddFileContractElement(fce types.FileContractElement) error {
	statement, err := tx.Prepare(`INSERT INTO elements(id, type, data) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := json.Marshal(fce)
	if err != nil {
		return err
	}
	_, err = statement.Exec(fce.ID.String(), "contract", data)
	return err
}

func (tx *Transaction) RemoveElement(id types.ElementID) error {
	statement, err := tx.Prepare(`DELETE FROM elements WHERE id=?`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(id.String())
	return err
}

func (tx *Transaction) AddChainStats(index types.ChainIndex, stats explorer.ChainStats) error {
	statement, err := tx.Prepare(`INSERT INTO chainstats(id, data) VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	_, err = statement.Exec(index.String(), data)
	return err
}

func (tx *Transaction) AddUnspentSiacoinElement(address types.Address, id types.ElementID) error {
	statement, err := tx.Prepare(`INSERT INTO unspentElements(address, type, id) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(address.String(), "siacoin", id.String())
	return err
}

func (tx *Transaction) AddUnspentSiafundElement(address types.Address, id types.ElementID) error {
	statement, err := tx.Prepare(`INSERT INTO unspentElements(address, type, id) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(address.String(), "siafund", id.String())
	return err
}

func (tx *Transaction) RemoveUnspentSiacoinElement(address types.Address, id types.ElementID) error {
	statement, err := tx.Prepare(`DELETE FROM unspentElements WHERE address=? AND id=? AND type=?`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(address.String(), id.String(), "siacoin")
	return err
}

func (tx *Transaction) RemoveUnspentSiafundElement(address types.Address, id types.ElementID) error {
	statement, err := tx.Prepare(`DELETE FROM unspentElements WHERE address=? AND id=? AND type=?`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(address.String(), id.String(), "siafund")
	return err
}

func (tx *Transaction) AddTransaction(txn types.Transaction, addresses []types.Address, block types.ChainIndex) error {
	statement, err := tx.Prepare(`INSERT INTO transactions(id, data) VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := json.Marshal(txn)
	if err != nil {
		return err
	}
	if _, err = statement.Exec(txn.ID().String(), data); err != nil {
		return err
	}

	for _, address := range addresses {
		statement, err := tx.Prepare(`INSERT INTO addressTransactions(address, id) VALUES(?, ?)`)
		if err != nil {
			return err
		}
		defer statement.Close()
		if _, err = statement.Exec(address.String(), txn.ID()); err != nil {
			return err
		}
	}
	return err
}
