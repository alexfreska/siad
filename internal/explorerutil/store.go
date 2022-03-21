package explorerutil

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strings"

	"go.sia.tech/core/types"
	"go.sia.tech/siad/v2/explorer"
	_ "modernc.org/sqlite"
)

type SqliteStore struct {
	db *sql.DB
	tx *sql.Tx
}

func NewStore(path string) (*SqliteStore, error) {
	db, err := sql.Open("sqlite", path)
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
	return &SqliteStore{db, nil}, nil
}

func (s *SqliteStore) CreateTransaction() error {
	if s.tx == nil {
		tx, err := s.db.BeginTx(context.Background(), nil)
		if err != nil {
			return err
		}
		s.tx = tx
	}
	return nil
}

func (s *SqliteStore) Commit() error {
	if err := s.tx.Commit(); err != nil {
		return err
	}
	s.tx = nil
	return nil
}

func decode(obj types.DecoderFrom, data []byte) error {
	d := types.NewDecoder(io.LimitedReader{R: bytes.NewReader(data), N: math.MaxInt64})
	obj.DecodeFrom(d)
	return d.Err()
}

func (s *SqliteStore) SiacoinElement(id types.ElementID) (sce types.SiacoinElement, err error) {
	row := s.tx.QueryRow(`SELECT data FROM elements WHERE id=? AND type=?`, id.String(), "siacoin")

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	err = decode(&sce, data)
	return
}

func (s *SqliteStore) SiafundElement(id types.ElementID) (sfe types.SiafundElement, err error) {
	row := s.tx.QueryRow(`SELECT data FROM elements WHERE id=? AND type=?`, id.String(), "siafund")

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	err = decode(&sfe, data)
	return
}

func (s *SqliteStore) FileContractElement(id types.ElementID) (fce types.FileContractElement, err error) {
	row := s.tx.QueryRow(`SELECT data FROM elements WHERE id=? AND type=?`, id.String(), "contract")

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	err = decode(&fce, data)
	return
}

func (s *SqliteStore) ChainStats(index types.ChainIndex) (cs explorer.ChainStats, err error) {
	row := s.tx.QueryRow(`SELECT data FROM chainstats WHERE id=?`, index.String())

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	err = decode(&cs, data)
	return
}

func parseElementID(str string) (id types.ElementID, err error) {
	err = id.UnmarshalText([]byte(str))
	return
}

func (s *SqliteStore) UnspentSiacoinElements(address types.Address) ([]types.ElementID, error) {
	rows, err := s.tx.Query(`SELECT id FROM unspentElements WHERE address=? AND type=?`, address.String(), "siacoin")
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
	rows, err := s.tx.Query(`SELECT id FROM unspentElements WHERE address=? AND type=?`, address.String(), "siafund")
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
	row := s.tx.QueryRow(`SELECT data FROM transactions WHERE id=?`, id.String())

	var data []byte
	if err = row.Scan(&data); err != nil {
		return
	}
	err = decode(&txn, data)
	return
}

func (s *SqliteStore) Transactions(address types.Address, amount int) ([]types.TransactionID, error) {
	rows, err := s.tx.Query(`SELECT id FROM addressTransactions WHERE address=? LIMIT 0,?`, address.String(), amount)
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

func encode(obj types.EncoderTo) ([]byte, error) {
	var buf bytes.Buffer
	e := types.NewEncoder(&buf)
	obj.EncodeTo(e)
	if err := e.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *SqliteStore) AddSiacoinElement(sce types.SiacoinElement) error {
	statement, err := s.tx.Prepare(`INSERT INTO elements(id, type, data) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := encode(sce)
	if err != nil {
		return err
	}
	_, err = statement.Exec(sce.ID.String(), "siacoin", data)
	return err
}

func (s *SqliteStore) AddSiafundElement(sfe types.SiafundElement) error {
	statement, err := s.tx.Prepare(`INSERT INTO elements(id, type, data) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := encode(sfe)
	if err != nil {
		return err
	}
	_, err = statement.Exec(sfe.ID.String(), "siafund", data)
	return err
}

func (s *SqliteStore) AddFileContractElement(fce types.FileContractElement) error {
	statement, err := s.tx.Prepare(`INSERT INTO elements(id, type, data) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := encode(fce)
	if err != nil {
		return err
	}
	_, err = statement.Exec(fce.ID.String(), "contract", data)
	return err
}

func (s *SqliteStore) RemoveElement(id types.ElementID) error {
	statement, err := s.tx.Prepare(`DELETE FROM elements WHERE id=?`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(id.String())
	return err
}

func (s *SqliteStore) AddChainStats(index types.ChainIndex, cs explorer.ChainStats) error {
	statement, err := s.tx.Prepare(`INSERT INTO chainstats(id, data) VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := encode(cs)
	if err != nil {
		return err
	}
	_, err = statement.Exec(index.String(), data)
	return err
}

func (s *SqliteStore) AddUnspentSiacoinElement(address types.Address, id types.ElementID) error {
	statement, err := s.tx.Prepare(`INSERT INTO unspentElements(address, type, id) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(address.String(), "siacoin", id.String())
	return err
}

func (s *SqliteStore) AddUnspentSiafundElement(address types.Address, id types.ElementID) error {
	statement, err := s.tx.Prepare(`INSERT INTO unspentElements(address, type, id) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(address.String(), "siafund", id.String())
	return err
}

func (s *SqliteStore) RemoveUnspentSiacoinElement(address types.Address, id types.ElementID) error {
	statement, err := s.tx.Prepare(`DELETE FROM unspentElements WHERE address=? AND id=? AND type=?`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(address.String(), id.String(), "siacoin")
	return err
}

func (s *SqliteStore) RemoveUnspentSiafundElement(address types.Address, id types.ElementID) error {
	statement, err := s.tx.Prepare(`DELETE FROM unspentElements WHERE address=? AND id=? AND type=?`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(address.String(), id.String(), "siafund")
	return err
}

func (s *SqliteStore) AddTransaction(txn types.Transaction, addresses []types.Address, block types.ChainIndex) error {
	statement, err := s.tx.Prepare(`INSERT INTO transactions(id, data) VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	data, err := encode(txn)
	if err != nil {
		return err
	}
	if _, err = statement.Exec(txn.ID().String(), data); err != nil {
		return err
	}

	for _, address := range addresses {
		statement, err := s.tx.Prepare(`INSERT INTO addressTransactions(address, id) VALUES(?, ?)`)
		if err != nil {
			return err
		}
		defer statement.Close()
		if _, err = statement.Exec(address.String(), txn.ID().String()); err != nil {
			return err
		}
	}
	return err
}
