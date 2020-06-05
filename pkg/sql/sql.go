package sql

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

const (
	DB = "gazelle"
)

type DCDb struct {
	DB *sql.DB
}

func OpenSql(user, pass, dbName string) (*DCDb, error) {
	db, err := sql.Open("mysql", fmt.Sprintf(`%s:%s@/%s`, user, pass, dbName))
	if err != nil {
		log.Panicln("failed opening db ", err.Error())
		return nil, err
	}
	log.Println("successfully open db :", db)
	return &DCDb{
		DB: db,
	}, nil
}

func (db *DCDb) GetEnabledPassKeys() ([]string, error) {
	keys := []string{}
	stmtOut, err := db.DB.Query("SELECT ID, can_leech, torrent_pass, (Visible='0' OR IP='127.0.0.1') AS Protected FROM users_main WHERE Enabled='1'")
	if err != nil {
		return keys, err
	}
	defer stmtOut.Close()

	var col1, col2, col3, col4 []byte
	for stmtOut.Next() {
		err = stmtOut.Scan(&col1, &col2, &col3, &col4)

		if err != nil {
			return keys, err
		}
		keys = append(keys, string(col3))
	}

	return keys, nil
}
