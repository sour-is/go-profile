package route

import (
	"database/sql"
	"sour.is/x/log"
	"net/http"
	"encoding/json"
)

func checktx(tx *sql.Tx, err error) {
	if err != nil {
		panic(err)
		log.Error("ROLLING BACK TRANSACTION")
		log.Error(err)
		tx.Rollback()
		return
	}

	if err = tx.Commit(); err != nil {
		log.Error(err)
		panic(err)
	}
}


func writeMsg(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(Message{code, msg})
}

func writeObject(w http.ResponseWriter, code int, o interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(o)
}

type Message struct {
	Code int    `json:"code"`
	Msg string  `json:"msg"`
}