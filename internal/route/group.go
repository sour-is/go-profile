package route

import (
	"net/http"
	"github.com/gorilla/mux"
	"sour.is/x/httpsrv"
	"sour.is/x/ident"
	"sour.is/x/profile/internal/model"
	"sour.is/x/dbm"
	"database/sql"
	"sour.is/x/profile/internal/profile"
)

func init() {
	httpsrv.IdentRegister("group", httpsrv.IdentRoutes{
		{"getGroupUsers",   "GET",    "/profile/group.users({aspect:[@0-9a-zA-Z._\\-\\*]+},{group:[@0-9a-zA-Z._\\-\\*]+})", getGroupUsers, },
		{"putGroupUser",    "PUT",    "/profile/group.user({aspect:[@0-9a-zA-Z._\\-\\*]+},{group:[@0-9a-zA-Z._\\-\\*]+},{user:[@0-9a-zA-Z._\\-\\*]+})", putGroupUser, },
		{"deleteGroupUser", "DELETE", "/profile/group.user({aspect:[@0-9a-zA-Z._\\-\\*]+},{group:[@0-9a-zA-Z._\\-\\*]+},{user:[@0-9a-zA-Z._\\-\\*]+})", deleteGroupUser, },

		{"getGroupRoles",   "GET",    "/profile/group.roles({aspect:[@:0-9a-zA-Z._\\-\\*]+},{group:[@:0-9a-zA-Z._\\-\\*]+})", getGroupRoles, },
		{"putGroupRole",    "PUT",    "/profile/group.role({assign_aspect:[@:0-9a-zA-Z._\\-\\*]+},{assign_group:[@:0-9a-zA-Z._\\-\\*]+},{aspect:[0-9a-zA-Z._\\-\\*]+},{role:[0-9a-zA-Z._\\-\\*]+})", putGroupRole, },
		{"deleteGroupRole", "DELETE", "/profile/group.role({assign_aspect:[@:0-9a-zA-Z._\\-\\*]+},{assign_group:[@:0-9a-zA-Z._\\-\\*]+},{aspect:[0-9a-zA-Z._\\-\\*]+},{role:[0-9a-zA-Z._\\-\\*]+})", deleteGroupRole, },

		{"getRoleGroups",   "GET",    "/profile/role.groups({aspect:[@:0-9a-zA-Z._\\-\\*]+},{role:[@:0-9a-zA-Z._\\-\\*]+})", getRoleGroups, },

		{"getUserRoles",        "GET", "/profile/user.roles({aspect:[@:0-9a-zA-Z._\\-\\*]+},{user:[@:0-9a-zA-Z._\\-\\*]+})", getUserRoles },
		{"getUserProfile",      "GET", "/profile/user.profile", getUserProfile },
		{"getUserOtherProfile", "GET", "/profile/user.profile({user:[@:0-9a-zA-Z._\\-\\*]+})", getUserProfile },
	})
}

func getGroupUsers (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	group := vars["group"]

	var lis []string

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		lis, err = model.GetGroupUserList(tx, aspect, group)
		return
	})

	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

    if len(lis) == 0 {
		writeMsg(w, http.StatusNotFound, "Not Found")
		return
	}

	writeObject(w, http.StatusOK, lis)
}
func putGroupUser (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	group := vars["group"]
	user := vars["user"]

	defer r.Body.Close()

	var allow bool
	var ok bool

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		// has group role?
		if allow, err = model.HasUserRole(tx, i.Aspect(), i.Identity(), "group+" + group, "owner", "admin"); err != nil {
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

		if ok, err = model.PutGroupUser(tx, aspect, group, user); err != nil {
			writeMsg(w, http.StatusConflict, err.Error())
			return
		}

		return
	})

	if err != nil {
		return
	}

	if !allow {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	if !ok {
		writeMsg(w, http.StatusNotModified, "Not Modified")
		return
	}

	writeObject(w, http.StatusCreated, "")
}
func deleteGroupUser (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	group := vars["group"]
	user := vars["user"]
	allow := false

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		// has group role?
		if allow, err = model.HasUserRole(tx, i.Aspect(), i.Identity(), "group+" + group, "owner", "admin"); err != nil {
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

		if err = model.DeleteGroupUser(tx, aspect, group, user); err != nil {
			writeMsg(w, http.StatusConflict, err.Error())
			return
		}

		return
	})

	if err != nil {
		return
	}

	if !allow {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	writeMsg(w, http.StatusNoContent, "No Content")
}

func getGroupRoles (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	group := vars["group"]

	var lis []string

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		lis, err = model.GetGroupRoleList(tx, aspect, group)
		return
	})

	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(lis) == 0 {
		writeMsg(w, http.StatusNotFound, "Not Found")

		return
	}

	writeObject(w, http.StatusOK, lis)
}
func putGroupRole (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	role := vars["role"]
	assignAspect := vars["assign_aspect"]
	assignGroup := vars["assign_group"]

	var ok bool
	var allow bool

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		// has role or owner?
		if allow, err = model.HasUserRole(tx, i.Aspect(), i.Identity(), "role+" + role, "owner", "admin"); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}

		// aspect must match auth
		if aspect != i.Aspect() {
			allow = false
		}

		// check if allowed.
		if !allow {
			return
		}

		if ok, err = model.PutRoleGroup(tx, aspect, role, assignAspect, assignGroup); err != nil {
			writeMsg(w, http.StatusConflict, err.Error())
			return
		}

		return
	})

	if err != nil {
		return
	}

	if !allow {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	if !ok {
		writeMsg(w, http.StatusNotModified, "Not Modified")
		return
	}

	writeMsg(w, http.StatusCreated, "Created")
}
func deleteGroupRole (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	role := vars["role"]
	assignAspect := vars["assign_aspect"]
	assignGroup := vars["assign_group"]

	allow := false
	err := dbm.Transaction(func(tx *sql.Tx) (err error) {

		// has role or owner?
		if allow, err = model.HasUserRole(tx, i.Aspect(), i.Identity(), "role+" + role, "owner", "admin"); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}

		// aspect must match auth
		if aspect != i.Aspect() {
			allow = false
			return
		}

		if !allow {
			writeMsg(w, http.StatusForbidden, "Access Denied")
			return
		}

		if err = model.DeleteRoleGroup(tx, aspect, role, assignAspect, assignGroup); err != nil {
			writeMsg(w, http.StatusConflict, err.Error())
			return
		}

		return
	})

	if err != nil {
		return
	}

	// check if allowed.
	if !allow {
		return
	}

	writeMsg(w, http.StatusNoContent, "No Content")
}

func getRoleGroups (w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	role := vars["role"]

	var lis []string

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		lis, err = model.GetRoleGroupList(tx, aspect, role)
		return
	})

	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(lis) == 0 {
		writeMsg(w, http.StatusNotFound, "Not Found")
		return
	}

	writeObject(w, http.StatusOK, lis)
}

func getUserRoles(w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	user := vars["user"]

	var lis []string

	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		lis, err = model.GetUserRoleList(tx, aspect, user)
		return
	})

	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(lis) == 0 {
		writeMsg(w, http.StatusNotFound, "Not Found")
		return
	}

	writeObject(w, http.StatusOK, lis)
}
func getUserProfile(w http.ResponseWriter, r *http.Request, i ident.Ident){
	vars := mux.Vars(r)
	aspect := i.Aspect()
	user := i.Identity()

	flag := profile.ProfileAll
	if u := vars["user"]; u != "" {
		user = vars["user"]
		flag = profile.ProfileOther
	}

	p, err := profile.GetUserProfile(aspect, user, flag)
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeObject(w, http.StatusOK, p)
}
