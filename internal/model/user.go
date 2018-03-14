package model

import (
	"database/sql"

	sq "gopkg.in/Masterminds/squirrel.v1"
	"sour.is/x/toolbox/dbm"
	"sour.is/x/toolbox/log"
)

type UserRole struct {
	User   string
	Aspect string
	Role   string
}

func HasUserRoleTx(tx *sql.Tx, aspect, user string, role ...string) (bool, error) {
	var ok int
	var err error

	err = sq.Select("count(*)").
		From("user_roles").
		Where(sq.Eq{
			"`aspect`": []string{"*", aspect},
			"`user`":   user,
			"`role`":   role}).
		RunWith(tx).QueryRow().Scan(&ok)

	if err != nil {
		log.Warning(err.Error())
		return false, err
	}

	log.Debugf("Count: %d, %+v", ok, role)

	return ok > 0, err
}
func HasUserRole(aspect, ident string, role ...string) (ok bool, err error) {
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		// has hash role?
		ok, err = HasUserRoleTx(
			tx,
			aspect,
			ident,
			role...)
		return
	})

	return
}

func GetUserRoles(tx *sql.Tx, aspect, user string) (lis []UserRole, err error) {

	var rows *sql.Rows
	rows, err = sq.Select("DISTINCT `role`").
		From("user_roles").
		Where(sq.Eq{"`aspect`": []string{"*", aspect}, "`user`": user}).
		RunWith(tx).Query()
	if err != nil {
		log.Debug(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var r UserRole
		if err = rows.Scan(&r.Role); err != nil {
			return
		}

		lis = append(lis, r)
	}

	return
}

func GetUserRoleList(tx *sql.Tx, aspect, user string) (lis []string, err error) {

	lis = make([]string, 0, 1)
	var roles []UserRole
	if roles, err = GetUserRoles(tx, aspect, user); err != nil {
		return
	}

	for _, r := range roles {
		lis = append(lis, r.Role)
	}

	return
}

type UserGroup struct {
	User   string
	Aspect string
	Group  string
}

func HasUserGroup(tx *sql.Tx, aspect, user string, role ...string) (bool, error) {

	var ok int
	var err error

	err = sq.Select("count(*)").
		From("group_users").
		Where(sq.Eq{
			"`aspect`": []string{"*", aspect},
			"`user`":   user,
			"`group`":  role}).
		RunWith(tx).QueryRow().Scan(&ok)

	if err != nil {
		log.Warning(err.Error())
		return false, err
	}

	log.Debugf("Count: %d, %+v", ok, role)

	return ok > 0, err
}

func GetUserGroups(tx *sql.Tx, aspect, user string) (lis []UserGroup, err error) {

	var rows *sql.Rows
	rows, err = sq.Select("DISTINCT `group`").
		From("group_users").
		Where(sq.Eq{"`aspect`": []string{"*", aspect}, "`user`": user}).
		RunWith(tx).Query()

	if err != nil {
		log.Debug(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var r UserGroup
		if err = rows.Scan(&r.Group); err != nil {
			return
		}

		lis = append(lis, r)
	}

	return
}

func GetUserGroupList(tx *sql.Tx, aspect, user string) (lis []string, err error) {

	lis = make([]string, 0, 1)
	var groups []UserGroup
	if groups, err = GetUserGroups(tx, aspect, user); err != nil {
		return
	}

	for _, r := range groups {
		lis = append(lis, r.Group)
	}

	return
}
