package route

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"sour.is/x/toolbox/dbm"
	"sour.is/x/toolbox/httpsrv"
	"sour.is/x/toolbox/ident"
	"sour.is/x/toolbox/log"
	"sour.is/x/profile/internal/model"
	"sour.is/x/profile/internal/profile"
)

func init() {
	httpsrv.IdentRegister("group", httpsrv.IdentRoutes{
		{"getAspects", "GET", "/v1/profile/aspect.list", getAspects},
		//		{"putAspect",           "PUT",    "/v1/profile/aspect({aspect:[@:0-9a-zA-Z._\\-\\*]+})",    putAspect },

		{"getGroups", "GET", "/v1/profile/aspect.group({aspect:[@:0-9a-zA-Z._\\-\\*]+})", getGroups},
		{"getGroupUsers", "GET", "/v1/profile/group.users({aspect:[@0-9a-zA-Z._\\-\\*]+},{group:[@0-9a-zA-Z._\\-\\*]+})", getGroupUsers},
		{"putGroupUser", "PUT", "/v1/profile/group.user({aspect:[@0-9a-zA-Z._\\-\\*]+},{group:[@0-9a-zA-Z._\\-\\*]+},{user:[@0-9a-zA-Z._\\-\\*]+})", putGroupUser},
		{"deleteGroupUser", "DELETE", "/v1/profile/group.user({aspect:[@0-9a-zA-Z._\\-\\*]+},{group:[@0-9a-zA-Z._\\-\\*]+},{user:[@0-9a-zA-Z._\\-\\*]+})", deleteGroupUser},

		{"getGroupRoles", "GET", "/v1/profile/group.roles({aspect:[@:0-9a-zA-Z._\\-\\*]+},{group:[@:0-9a-zA-Z._\\-\\*]+})", getGroupRoles},
		{"putGroupRole", "PUT", "/v1/profile/group.role({assign_aspect:[@:0-9a-zA-Z._\\-\\*]+},{assign_group:[@:0-9a-zA-Z._\\-\\*]+},{aspect:[@:0-9a-zA-Z._\\-\\*]+},{role:[@:0-9a-zA-Z._\\-\\*]+})", putGroupRole},
		{"deleteGroupRole", "DELETE", "/v1/profile/group.role({assign_aspect:[@:0-9a-zA-Z._\\-\\*]+},{assign_group:[@:0-9a-zA-Z._\\-\\*]+},{aspect:[@:0-9a-zA-Z._\\-\\*]+},{role:[@:0-9a-zA-Z._\\-\\*]+})", deleteGroupRole},

		{"getRoleGroups", "GET", "/v1/profile/role.groups({aspect:[@:0-9a-zA-Z._\\-\\*]+},{role:[@:0-9a-zA-Z._\\-\\*]+})", getRoleGroups},
		{"getUserRoles", "GET", "/v1/profile/user.roles({aspect:[@:0-9a-zA-Z._\\-\\*]+},{user:[@:0-9a-zA-Z._\\-\\*]+})", getUserRoles},

		{"getUserProfile", "GET", "/v1/profile/user.profile", getUserProfile},
		{"getUserOtherProfile", "GET", "/v1/profile/user.profile({user:[@:0-9a-zA-Z._\\-\\*]+})", getUserProfile},
		{"putUserProfile", "PUT", "/v1/profile/user.profile", putUserProfile},
		{"putUserOtherProfile", "PUT", "/v1/profile/user.profile({user:[@:0-9a-zA-Z._\\-\\*]+})", putUserProfile},
	})
}

func getAspects(w httpsrv.ResponseWriter, _ *http.Request, i ident.Ident) {
	var lis []string

	var allow bool
	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		// has admin role?
		if allow, err = model.HasUserRoleTx(tx, i.GetAspect(), i.GetIdentity(), "admin"); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !allow {
			lis, err = model.GetAspectList(tx, false)
			return
		}

		lis, err = model.GetAspectList(tx, true)
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
func getGroups(w httpsrv.ResponseWriter, r *http.Request, _ ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]

	var lis []string

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		lis, err = model.GetGroupList(tx, aspect)
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

func getGroupUsers(w httpsrv.ResponseWriter, r *http.Request, _ ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	group := vars["group"]

	var lis []string

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
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
func putGroupUser(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	group := vars["group"]
	user := vars["user"]

	defer r.Body.Close()

	var allow bool
	var ok bool

	// aspect must match auth
	if aspect != i.GetAspect() {
		writeMsg(w, http.StatusForbidden, "Aspect should match "+aspect)
		return
	}

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		// has group role?
		if allow, err = model.HasUserRoleTx(tx, i.GetAspect(), i.GetIdentity(), "group+"+group, "owner", "admin"); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
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
func deleteGroupUser(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	group := vars["group"]
	user := vars["user"]
	allow := false

	// / aspect must match auth
	if aspect != i.GetAspect() {
		writeMsg(w, http.StatusForbidden, "Aspect should match "+aspect)
		return
	}

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		// has group role?
		if allow, err = model.HasUserRoleTx(tx, i.GetAspect(), i.GetIdentity(), "group+"+group, "owner", "admin"); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
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

func getGroupRoles(w httpsrv.ResponseWriter, r *http.Request, _ ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	group := vars["group"]

	var lis []string

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
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
func putGroupRole(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	role := vars["role"]
	assignAspect := vars["assign_aspect"]
	assignGroup := vars["assign_group"]

	var ok bool
	var allow bool

	// aspect must match auth
	if aspect != i.GetAspect() {
		writeMsg(w, http.StatusForbidden, "Aspect should match "+aspect)
		return
	}

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		// has role or owner?
		if allow, err = model.HasUserRoleTx(tx, i.GetAspect(), i.GetIdentity(), "role+"+role, "owner", "admin"); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
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
func deleteGroupRole(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	role := vars["role"]
	assignAspect := vars["assign_aspect"]
	assignGroup := vars["assign_group"]

	log.Debugf("Delete %s/%s : %s/%s", assignAspect, assignGroup, aspect, role)

	allow := false

	// aspect must match auth
	if aspect != i.GetAspect() {
		writeMsg(w, http.StatusForbidden, "Aspect should match "+aspect)
		return
	}

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
		// has role or owner?
		if allow, err = model.HasUserRoleTx(tx, i.GetAspect(), i.GetIdentity(), "role+"+role, "owner", "admin"); err != nil {
			writeMsg(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !allow {
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
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	writeMsg(w, http.StatusNoContent, "No Content")
}

func getRoleGroups(w httpsrv.ResponseWriter, r *http.Request, _ ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	role := vars["role"]

	var lis []string

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
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
func getUserRoles(w httpsrv.ResponseWriter, r *http.Request, _ ident.Ident) {
	vars := mux.Vars(r)
	aspect := vars["aspect"]
	user := vars["user"]

	var lis []string

	err := dbm.Transaction(func(tx *dbm.Tx) (err error) {
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

func getUserProfile(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	aspect := i.GetAspect()
	user := i.GetIdentity()

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
func putUserProfile(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	var err error

	vars := mux.Vars(r)
	aspect := i.GetAspect()
	user := i.GetIdentity()

	flag := profile.ProfileAll
	if u := vars["user"]; u != "" {
		user = vars["user"]
		flag = profile.ProfileOther
	}

	defer r.Body.Close()

	var p profile.Profile
	if err = json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	var ok bool

	if user == i.GetIdentity() {
		ok = true
	}
	if !ok {
		ok = i.HasRole("admin")
	}
	if !ok {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	err = profile.PutUserProfile(aspect, user, p)
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	p, err = profile.GetUserProfile(aspect, user, flag)
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeObject(w, http.StatusCreated, p)
}
