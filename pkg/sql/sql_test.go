package sql

import (
	"log"
	"testing"
)

func TestOpen(t *testing.T) {
	mDB, err := OpenSql(`root`, `y9#Z';r"`, `gazelle`)
	if err != nil {
		t.Errorf("Open failed %s", err.Error())
	}

	keys, err := mDB.GetEnabledPassKeys()
	if err != nil {
		t.Errorf("Scan failed %s", err.Error())
	}

	if len(keys) == 0 {
		t.Errorf("Keys should not be zero")
	}

	log.Println(keys)
}
