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
	"bytes"
)

func init() {
	httpsrv.IdentRegister("peer", httpsrv.IdentRoutes{
		{"getNodes", "GET", "/v1/peers/peer.nodes", getNodes},
		{"postNode", "POST", "/v1/peers/peer.nodes", putNode},
		{"getNode", "GET", "/v1/peers/peer.node({id})", getNode},
		{"putNode", "PUT", "/v1/peers/peer.node({id})", putNode},
		{"putNode", "DELETE", "/v1/peers/peer.node({id})", deleteNode},

		{"getRegIndex", "GET", "/v1/reg/reg.index({name:[0-9A-Z\\-]+})", getRegIndex},
		{"getRegObject", "GET", "/v1/reg/reg.object({type},{name:[0-9a-zA-Z\\-\\.:_]+})", getRegObject},
		{"getRegObjects", "GET", "/v1/reg/reg.objects", getRegObjects},

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
	if node.Owner == "" {
		node.Owner = strings.ToLower(i.Identity())
	}

	if strings.ToLower(node.Owner) != strings.ToLower(i.Identity()) {
		writeMsg(w, http.StatusForbidden, "peer_owner should match user ident "+node.Owner+" == "+i.Identity())
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
		if strings.ToLower(check.Owner) != strings.ToLower(i.Identity()) {
			ok = false
			writeMsg(w, http.StatusForbidden, "peer_owner should match user ident "+node.Owner+" == "+i.Identity())
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
		if strings.ToLower(node.Owner) != i.Identity() {
			ok = false
			writeMsg(w, http.StatusForbidden, "peer_owner should match user ident: "+node.Owner+" == "+i.Identity())
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

func getRegIndex(w http.ResponseWriter, r *http.Request, _ ident.Ident) {
	vars := mux.Vars(r)
	id := vars["name"]

	var err error
	var lis []model.RegIndex
/*
	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}
*/
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		lis, err = model.GetRegIndex(tx, id)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	var m [][]string
	for _, n := range lis {
		m = append(m, []string{n.Type, n.Name})
	}

	writeObject(w, http.StatusOK, m)
}

func getRegObject(w http.ResponseWriter, r *http.Request, _ ident.Ident) {
	vars := mux.Vars(r)
	typeId := vars["type"]
	name := vars["name"]

	var err error
	var o model.RegObject

/*
	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}
*/
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		o, err = model.GetRegObject(tx, typeId, name)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	var m [][]string
	for _, n := range o.Items {
		m = append(m, []string{n.Field, n.Value})
	}

	writeObject(w, http.StatusOK, m)
}


func getRegObjects(w http.ResponseWriter, r *http.Request, _ ident.Ident) {

	var err error
	var lis []model.RegObject

	/*
		if !i.LoggedIn() {
			writeMsg(w, http.StatusForbidden, "Access Denied")
			return
		}
	*/
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		lis, err = model.GetRegObjects(tx, r.URL.Query().Get("filter"), r.URL.Query().Get("fields"))
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}



	if strings.Contains(r.Header.Get("accept"),"application/json") {
		m := make([][][]string,0,len(lis))
		for _, o := range lis {
			var l [][]string
			for _, n := range o.Items {
				l = append(l, []string{n.Field, n.Value})
			}
			m = append(m, l)
		}

		writeObject(w, http.StatusOK, m)
	} else {
		buf := new(bytes.Buffer)
		for _, o := range lis {
			buf.WriteString("---\n")
			for _, n := range o.Items {
				buf.WriteString(n.Field)
				buf.WriteString(": ")
				buf.WriteString(n.Value)
				buf.WriteString("\n")
			}
		}

		writeText(w, http.StatusOK, buf.String())
	}
}
