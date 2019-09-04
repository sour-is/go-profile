package route

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"sour.is/x/profile/internal/model"
	"sour.is/x/toolbox/dbm"
	"sour.is/x/toolbox/httpsrv"
	"sour.is/x/toolbox/ident"
)

func init() {
	httpsrv.IdentRegister("hash", httpsrv.IdentRoutes{
		{"getHashList", "GET", "/v1/profile/hash.list({aspect:[@:0-9a-zA-Z._\\-\\*]+})", getHashList},
		{"getHash", "GET", "/v1/profile/hash.map({aspect:[@:0-9a-zA-Z._\\-\\*]+},{name:[@:0-9a-zA-Z._\\-\\*]+})", getHash},
		{"putHash", "PUT", "/v1/profile/hash.map({aspect:[@:0-9a-zA-Z._\\-\\*]+},{name:[@:0-9a-zA-Z._\\-\\*]+})", putHash},
		{"deleteHash", "DELETE", "/v1/profile/hash.map({aspect:[@:0-9a-zA-Z._\\-\\*]+},{name:[@:0-9a-zA-Z._\\-\\*]+})", deleteHash},
	})
}

func getHashList(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]

	var lis []string
	var allow bool

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		// has group role?
		if allow, err = model.HasUserRoleTx(tx, i.GetAspect(), i.GetIdentity(), "owner", "admin"); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}
		// aspect must match auth
		if aspect != i.GetAspect() {
			allow = false
		}
		if !allow {
			return
		}

		lis, err = model.GetHashList(tx, aspect)
		return
	})

	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !allow {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	if len(lis) == 0 {
		writeMsg(w, http.StatusNotFound, "Not Found")
		return
	}

	writeObject(w, http.StatusOK, lis)
}
func getHash(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	name := vars["name"]

	var ok bool
	var allow bool
	var m map[string]string

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		if ok, err = model.HasUserRoleTx(
			tx,
			i.GetAspect(),
			i.GetIdentity(),
			"hash.read."+name,
			"hash.write."+name,
			"hash.reader",
			"hash.writer",
			"owner",
			"admin"); ok {
			allow = true
		}
		if err != nil {
			return
		}

		// aspect must match auth
		if aspect != i.GetAspect() {
			allow = false
			return
		}

		m, ok, err = model.GetHashMap(tx, aspect, name)
		if err != nil {
			return
		}

		return
	})

	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !allow {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	if !ok {
		writeMsg(w, http.StatusNotFound, "Not Found")
		return
	}

	writeObject(w, http.StatusOK, m)
}
func putHash(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	name := vars["name"]

	defer r.Body.Close()

	var err error
	var keys map[string]string
	if err = json.NewDecoder(r.Body).Decode(&keys); err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	var allow bool
	var m map[string]string

	err = dbm.Transaction(func(tx *dbm.Tx) (err error) {
		// has hash role?
		allow, err = model.HasUserRoleTx(
			tx,
			i.GetAspect(),
			i.GetIdentity(),
			"hash.write."+name,
			"hash.writer",
			"owner",
			"admin")
		if err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}

		// aspect must match auth
		if aspect != i.GetAspect() {
			allow = false
		}
		if !allow {
			return
		}

		if err = model.PutHashMap(tx, aspect, name, keys); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}

		m, _, err = model.GetHashMap(tx, aspect, name)

		return
	})

	if err != nil {
		return
	}

	if !allow {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	if len(m) == 0 {
		writeObject(w, http.StatusNoContent, m)
		return
	}

	writeObject(w, http.StatusCreated, m)
}
func deleteHash(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	name := vars["name"]

	defer r.Body.Close()

	var allow bool

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		// has hash role?
		allow, err = model.HasUserRoleTx(
			tx,
			i.GetAspect(),
			i.GetIdentity(),
			"hash.write."+name,
			"hash.writer",
			"owner",
			"admin")
		if err != nil {
			return
		}
		// aspect must match auth
		if aspect != i.GetAspect() {
			allow = false
		}
		if !allow {
			return
		}

		if _, err = model.DeleteHashMap(tx, aspect, name); err != nil {
			return
		}

		return
	})

	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !allow {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	writeMsg(w, http.StatusNoContent, "No Content")
}
