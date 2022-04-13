package explorerutil

import (
	"bytes"
	"context"
	"database/sql"
	"strings"

	"go.sia.tech/core/consensus"
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
    id BINARY(128) PRIMARY KEY,
    type BINARY(128),
    data BLOB NOT NULL
);

CREATE TABLE validationContexts (
    id BINARY(128) PRIMARY KEY,
    data BLOB NOT NULL
);

CREATE TABLE chainstats (
    id BINARY(128) PRIMARY KEY,
    data BLOB NOT NULL
);

CREATE TABLE unspentElements (
    id BINARY(128) PRIMARY KEY,
    type BINARY(128),
    address BINARY(128)
);

CREATE TABLE transactions (
    id BINARY(128) PRIMARY KEY,
    data BLOB NOT NULL
);

CREATE TABLE addressTransactions (
    id BINARY(128),
    address BINARY(128)
);
`); err != nil && !strings.Contains(err.Error(), "already exists") {
		return nil, err
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
	d := types.NewBufDecoder(data)
	obj.DecodeFrom(d)
	return d.Err()
}

func (s *SqliteStore) queryRow(d types.DecoderFrom, query string, args ...interface{}) (err error) {
	row := s.tx.QueryRow(query, args...)
	var data []byte
	if err = row.Scan(&data); err == nil {
		err = decode(d, data)
	}
	return
}

func (s *SqliteStore) SiacoinElement(id types.ElementID) (sce types.SiacoinElement, err error) {
	err = s.queryRow(&sce, `SELECT data FROM elements WHERE id=? AND type=?`, encode(id), "siacoin")
	return
}

func (s *SqliteStore) SiafundElement(id types.ElementID) (sfe types.SiafundElement, err error) {
	err = s.queryRow(&sfe, `SELECT data FROM elements WHERE id=? AND type=?`, encode(id), "siafund")
	return
}

func (s *SqliteStore) FileContractElement(id types.ElementID) (fce types.FileContractElement, err error) {
	err = s.queryRow(&fce, `SELECT data FROM elements WHERE id=? AND type=?`, encode(id), "contract")
	return
}

func (s *SqliteStore) ChainStats(index types.ChainIndex) (cs explorer.ChainStats, err error) {
	err = s.queryRow(&cs, `SELECT data FROM chainstats WHERE id=?`, index.String())
	return
}

func (s *SqliteStore) UnspentSiacoinElements(address types.Address) ([]types.ElementID, error) {
	rows, err := s.tx.Query(`SELECT id FROM unspentElements WHERE address=? AND type=?`, encode(address), "siacoin")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []types.ElementID
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var id types.ElementID
		if err := decode(&id, data); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SqliteStore) UnspentSiafundElements(address types.Address) ([]types.ElementID, error) {
	rows, err := s.tx.Query(`SELECT id FROM unspentElements WHERE address=? AND type=?`, encode(address), "siafund")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []types.ElementID
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var id types.ElementID
		if err := decode(&id, data); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SqliteStore) Transaction(id types.TransactionID) (txn types.Transaction, err error) {
	err = s.queryRow(&txn, `SELECT data FROM transactions WHERE id=?`, encode(id))
	return
}

func (s *SqliteStore) Transactions(address types.Address, amount, offset int) ([]types.TransactionID, error) {
	rows, err := s.tx.Query(`SELECT id FROM addressTransactions WHERE address=? LIMIT ? OFFSET ?`, encode(address), amount, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []types.TransactionID
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var id types.TransactionID
		if err := decode(&id, data); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SqliteStore) ValidationContext(index types.ChainIndex) (context consensus.ValidationContext, err error) {
	err = s.queryRow(&context, `SELECT data FROM validationContexts WHERE id=?`, encode(index))
	return
}

func encode(obj types.EncoderTo) []byte {
	var buf bytes.Buffer
	e := types.NewEncoder(&buf)
	obj.EncodeTo(e)
	e.Flush()
	return buf.Bytes()
}

func (s *SqliteStore) execStatement(statement string, args ...interface{}) error {
	stmt, err := s.tx.Prepare(statement)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(args...)
	return err
}

func (s *SqliteStore) AddSiacoinElement(sce types.SiacoinElement) error {
	return s.execStatement(`INSERT INTO elements(id, type, data) VALUES(?, ?, ?)`, encode(sce.ID), "siacoin", encode(sce))
}

func (s *SqliteStore) AddSiafundElement(sfe types.SiafundElement) error {
	return s.execStatement(`INSERT INTO elements(id, type, data) VALUES(?, ?, ?)`, encode(sfe.ID), "siafund", encode(sfe))
}

func (s *SqliteStore) AddFileContractElement(fce types.FileContractElement) error {
	return s.execStatement(`INSERT INTO elements(id, type, data) VALUES(?, ?, ?)`, encode(fce.ID), "contract", encode(fce))
}

func (s *SqliteStore) RemoveElement(id types.ElementID) error {
	return s.execStatement(`DELETE FROM elements WHERE id=?`, encode(id))
}

func (s *SqliteStore) AddChainStats(index types.ChainIndex, cs explorer.ChainStats) error {
	return s.execStatement(`INSERT INTO chainstats(id, data) VALUES(?, ?)`, index.String(), encode(cs))
}

func (s *SqliteStore) AddUnspentSiacoinElement(address types.Address, id types.ElementID) error {
	return s.execStatement(`INSERT INTO unspentElements(address, type, id) VALUES(?, ?, ?)`, encode(address), "siacoin", encode(id))
}

func (s *SqliteStore) AddUnspentSiafundElement(address types.Address, id types.ElementID) error {
	return s.execStatement(`INSERT INTO unspentElements(address, type, id) VALUES(?, ?, ?)`, encode(address), "siafund", encode(id))
}

func (s *SqliteStore) RemoveUnspentSiacoinElement(address types.Address, id types.ElementID) error {
	return s.execStatement(`DELETE FROM unspentElements WHERE address=? AND id=? AND type=?`, encode(address), encode(id), "siacoin")
}

func (s *SqliteStore) RemoveUnspentSiafundElement(address types.Address, id types.ElementID) error {
	return s.execStatement(`DELETE FROM unspentElements WHERE address=? AND id=? AND type=?`, encode(address), encode(id), "siafund")
}

func (s *SqliteStore) AddTransaction(txn types.Transaction, addresses []types.Address, block types.ChainIndex) error {
	id := encode(txn.ID())
	if err := s.execStatement(`INSERT INTO transactions(id, data) VALUES(?, ?)`, id, encode(txn)); err != nil {
		return err
	}

	for _, address := range addresses {
		if err := s.execStatement(`INSERT INTO addressTransactions(address, id) VALUES(?, ?)`, encode(address), id); err != nil {
			return err
		}
	}
	return nil
}

func (s *SqliteStore) AddValidationContext(index types.ChainIndex, context consensus.ValidationContext) error {
	return s.execStatement(`INSERT INTO validationContexts(id, data) VALUES(?, ?)`, encode(index), encode(context))
}
