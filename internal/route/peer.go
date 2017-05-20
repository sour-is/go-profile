package route

import (
	"database/sql"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"sour.is/x/dbm"
	"sour.is/x/httpsrv"
	"sour.is/x/ident"
	"sour.is/x/profile/internal/model"
	"strings"
)

func init() {
	httpsrv.IdentRegister("peer", httpsrv.IdentRoutes{
		{"getNodes", "GET", "/v1/peers/peer.nodes", getNodes},
		{"postNode", "POST", "/v1/peers/peer.nodes", putNode},
		{"getNode", "GET", "/v1/peers/peer.node({id})", getNode},
		{"putNode", "PUT", "/v1/peers/peer.node({id})", putNode},
		{"putNode", "DELETE", "/v1/peers/peer.node({id})", deleteNode},
	})
}

func getNodes(w http.ResponseWriter, _ *http.Request, i ident.Ident) {
	var lis []model.PeerNode

	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}
	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		lis, err = model.GetPeerList(tx, i.Identity())
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeObject(w, http.StatusOK, lis)
}
func getNode(w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	id := vars["id"]

	defer r.Body.Close()

	var ok bool
	var err error
	var node model.PeerNode

	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		node, ok, err = model.GetPeerNode(tx, id, false)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeMsg(w, http.StatusNotFound, "Not Found")
		return
	}

	writeObject(w, http.StatusOK, node)
}
func putNode(w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	id := vars["id"]

	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	var err error
	var node model.PeerNode
	if err = json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	node.Id = id
	if strings.ToLower(node.Nick) != i.Identity() {
		writeMsg(w, http.StatusForbidden, "peer_nick should match user ident "+node.Nick+" == "+i.Identity())
		return
	}

	var ok bool
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		var check model.PeerNode
		if check, ok, err = model.GetPeerNode(tx, id, true); err != nil {
			return
		}
		if !ok {
			ok = true
			node, err = node.Insert(tx)

			return
		}
		if check.Nick != i.Identity() {
			ok = false
			writeMsg(w, http.StatusForbidden, "peer_nick should match user ident "+node.Nick+" == "+i.Identity())
			return
		}
		if node, err = node.Update(tx); err != nil {
			return
		}

		return
	})
	if err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}
	if !ok {
		return
	}

	writeObject(w, http.StatusCreated, node)
}
func deleteNode(w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	id := vars["id"]

	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	var ok bool
	var err error
	var node model.PeerNode

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		node, ok, err = model.GetPeerNode(tx, id, true)
		if !ok {
			writeMsg(w, http.StatusNotFound, "Not Found")
			return
		}
		if strings.ToLower(node.Nick) != i.Identity() {
			ok = false
			writeMsg(w, http.StatusForbidden, "peer_nick should match user ident: "+node.Nick+" == "+i.Identity())
			return
		}

		ok = true
		model.DeletePeerNode(tx, id)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}
	if !ok {
		return
	}

	writeObject(w, http.StatusNoContent, node)
}
