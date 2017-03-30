package model

import (
	"sour.is/x/log"
	sq "gopkg.in/Masterminds/squirrel.v1"
	"database/sql"
)

type UserRole struct {
	User string
	Aspect string
	Role string
}

func HasUserRole(tx *sql.Tx, aspect, user string, role ...string) (bool, error) {

	var ok int
	var err error

	err = sq.Select("count(*)").
		From("user_roles").
		Where(sq.Eq{
			"`aspect`": []string{"*", aspect},
			"`user`": user,
			"`role`": role}).
		RunWith(tx).QueryRow().Scan(&ok)

	if err != nil {
		log.Warning(err.Error())
		return false, err
	}

	log.Debugf("Count: %d, %+v", ok, role)

	return ok > 0, err
}

func GetUserRoles(tx *sql.Tx, aspect, user string) (lis []UserRole, err error){

	var rows *sql.Rows
	rows, err = sq.Select("`aspect`", "`user`", "`role`").
		From("user_roles").
		Where(sq.Eq{"`aspect`": []string{"*", aspect}, "`user`": user}).
		RunWith(tx).Query()

	if err != nil {
		log.Println(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var r UserRole
		if err = rows.Scan(&r.Aspect, &r.User, &r.Role); err != nil {
			return
		}

		lis = append(lis, r)
	}

	return
}

func GetUserRoleList(tx *sql.Tx, aspect, user string) (lis []string, err error) {

	var roles []UserRole
	if roles, err = GetUserRoles(tx, aspect, user); err != nil {
		return
	}

	for _, r := range roles {
		lis = append(lis, r.Role)
	}

	return
}


