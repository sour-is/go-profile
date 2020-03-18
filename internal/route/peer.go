package route

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"

	"sour.is/x/profile/internal/model"
	"sour.is/x/toolbox/dbm"
	"sour.is/x/toolbox/httpsrv"
	"sour.is/x/toolbox/ident"
)

var bm *bluemonday.Policy

func init() {
	httpsrv.IdentRegister("peer", httpsrv.IdentRoutes{
		{"getNodes", "GET", "/v1/peers/peer.nodes", getNodes},
		{"postNode", "POST", "/v1/peers/peer.nodes", putNode},
		{"getNode", "GET", "/v1/peers/peer.node({id})", getNode},
		{"putNode", "PUT", "/v1/peers/peer.node({id})", putNode},
		{"putNode", "DELETE", "/v1/peers/peer.node({id})", deleteNode},
	})

	bm = bluemonday.UGCPolicy()
}

func getNodes(w httpsrv.ResponseWriter, _ *http.Request, i ident.Ident) {
	var lis []model.PeerNode

	if !i.IsActive() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}
	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		lis, err = model.GetPeerList(tx, i.GetIdentity())
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeObject(w, http.StatusOK, lis)
}
func getNode(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	id := vars["id"]

	defer r.Body.Close()

	var ok bool
	var err error
	var node model.PeerNode

	if !i.IsActive() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	err = dbm.Transaction(func(tx *dbm.Tx) (err error) {
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
func putNode(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	id := vars["id"]

	if !i.IsActive() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	var err error
	var node model.PeerNode
	if err = json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	node.Name = bm.Sanitize(node.Name)
	node.Note = bm.Sanitize(node.Note)
	node.Country = bm.Sanitize(node.Country)
	node.Nick = bm.Sanitize(node.Nick)
	node.Type = bm.Sanitize(node.Type)

	node.Id = id
	if node.Owner == "" {
		node.Owner = strings.ToLower(i.GetIdentity())
	}

	if strings.ToLower(node.Owner) != strings.ToLower(i.GetIdentity()) {
		writeMsg(w, http.StatusForbidden, "peer_owner should match user ident "+node.Owner+" == "+i.GetIdentity())
		return
	}

	var ok bool
	err = dbm.Transaction(func(tx *dbm.Tx) (err error) {
		var check model.PeerNode
		if check, ok, err = model.GetPeerNode(tx, id, true); err != nil {
			return
		}
		if !ok {
			ok = true
			node, err = node.Insert(tx)

			return
		}
		if strings.ToLower(check.Owner) != strings.ToLower(i.GetIdentity()) {
			ok = false
			writeMsg(w, http.StatusForbidden, "peer_owner should match user ident "+node.Owner+" == "+i.GetIdentity())
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
func deleteNode(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	id := vars["id"]

	if !i.IsActive() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	var ok bool
	var err error
	var node model.PeerNode

	err = dbm.Transaction(func(tx *dbm.Tx) (err error) {
		node, ok, err = model.GetPeerNode(tx, id, true)
		if !ok {
			writeMsg(w, http.StatusNotFound, "Not Found")
			return
		}
		if strings.ToLower(node.Owner) != i.GetIdentity() {
			ok = false
			writeMsg(w, http.StatusForbidden, "peer_owner should match user ident: "+node.Owner+" == "+i.GetIdentity())
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
