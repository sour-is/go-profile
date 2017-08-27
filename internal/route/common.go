package route

import (
	"encoding/json"
	"net/http"
)

type Message struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
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
func writeText(w http.ResponseWriter, code int, o string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	w.Write([]byte(o))
}
