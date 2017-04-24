package route

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"sour.is/x/httpsrv"
	"sour.is/x/ident"
	"sour.is/x/profile/internal/model"
	"sour.is/x/dbm"
	"database/sql"
)

func init() {
	httpsrv.IdentRegister("hash", httpsrv.IdentRoutes{
		{"getHashList", "GET",    "/profile/hash.list({aspect:[@:0-9a-zA-Z._\\-\\*]+})", getHashList, },
		{"getHash",     "GET",    "/profile/hash.map({aspect:[@:0-9a-zA-Z._\\-\\*]+},{name:[@:0-9a-zA-Z._\\-\\*]+})", getHash, },
		{"putHash",     "PUT",    "/profile/hash.map({aspect:[@:0-9a-zA-Z._\\-\\*]+},{name:[@:0-9a-zA-Z._\\-\\*]+})", putHash, },
		{"deleteHash",  "DELETE", "/profile/hash.map({aspect:[@:0-9a-zA-Z._\\-\\*]+},{name:[@:0-9a-zA-Z._\\-\\*]+})", deleteHash, },
	})
}

func getHashList (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]

	var lis []string
	var allow bool

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		// has group role?
		if allow, err = model.HasUserRoleTx(tx, i.Aspect(), i.Identity(), "owner", "admin"); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}
		// aspect must match auth
		if aspect != i.Aspect() {
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
func getHash (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	name := vars["name"]

	var ok bool
	var allow bool
	var m map[string]string

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		if ok, err = model.HasUserRoleTx(
			tx,
			i.Aspect(),
			i.Identity(),
			"hash.read." + name,
			"hash.write." + name,
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
		if aspect != i.Aspect() {
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
func putHash (w http.ResponseWriter, r *http.Request, i ident.Ident) {
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

	var ok bool
	var allow bool
	var m map[string]string

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		// has hash role?
		allow, err = model.HasUserRoleTx(
			tx,
			i.Aspect(),
			i.Identity(),
			"hash.write." + name,
			"hash.writer",
			"owner",
			"admin")
		if err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}

		// aspect must match auth
		if aspect != i.Aspect() {
			allow = false
		}
		if !allow {
			return
		}

		if err = model.PutHashMap(tx, aspect, name, keys); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}

		m, ok, err = model.GetHashMap(tx, aspect, name);

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
func deleteHash (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	name := vars["name"]

	defer r.Body.Close()

	var allow bool

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		// has hash role?
		allow, err = model.HasUserRoleTx(
			tx,
			i.Aspect(),
			i.Identity(),
			"hash.write." + name,
			"hash.writer",
			"owner",
			"admin")
		if err != nil {
			return
		}
		// aspect must match auth
		if aspect != i.Aspect() {
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
