package model

import (
	"sour.is/x/log"
	sq "gopkg.in/Masterminds/squirrel.v1"
	"database/sql"
)

type GroupUser struct {
	GroupId int
	Aspect string
	Group string
	User string
}
type GroupRole struct {
	GroupId int
	Aspect string
	Role string
	AssignAspect string
	AssignGroup string
}

func HasGroup(tx *sql.Tx, aspect, group string) (bool, error){
	var id int
	var err error

	err = sq.Select("count(*)").
		From("`group`").
		Where(sq.Eq{
		"`aspect`": aspect,
		"`group`": group}).
		RunWith(tx).QueryRow().Scan(&id)

	if err != nil {
		log.Print(err.Error())
		return false, err
	}

	return id > 0, err
}

func GetGroupId(tx *sql.Tx, aspect, group string, lock bool) (int, error){
	var id int
	var err error

	s := sq.Select("`group_id`").
		From("`group`").
		Where(sq.Eq{
			"`aspect`": aspect,
			"`group`": group})

	if lock {
		s.Suffix("LOCK IN SHARE MODE")
	}

	err = s.RunWith(tx).QueryRow().Scan(&id)

	if err != nil {
		log.Print(err.Error())
		return 0, err
	}

	return id, err
}

func PutGroupId(tx *sql.Tx, aspect, group string) (int, error){
	var id int64
	var err error

	var res sql.Result
	res, err = sq.Insert("`group`").
		Columns("`aspect`", "`group`").
		Values(aspect, group).
		RunWith(tx).Exec()

	if err != nil {
		log.Print(err.Error())
		return 0, err
	}

	id, err = res.LastInsertId()

	return int(id), err
}

func HasGroupUser(tx *sql.Tx, group_id int, user string, lock bool) (bool, error) {

	var ok int
	var err error

	s := sq.Select("count(*)").
		From("`group_user`").
		Where(sq.Eq{
		"`group_id`": group_id,
		"`user`": user})

	if lock {
		s.Suffix("FOR UPDATE")
	}

	s.RunWith(tx).QueryRow().Scan(&ok)

	if err != nil {
		log.Print(err.Error())
		return false, err
	}

	log.Printf("Count: %d", ok)

	return ok > 0, err
}

func GetGroupUsers(tx *sql.Tx, group_id int) (users []GroupUser, err error) {
	var rows *sql.Rows

	rows, err = sq.Select("`group_id`", "`aspect`", "`group`", "`user`").
		From("`group_users`").
		Where(sq.Eq{"`group_id`": group_id}).
		RunWith(tx).Query()

	if err != nil {
		log.Println(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var u GroupUser
		if err = rows.Scan(&u.GroupId, &u.Aspect, &u.Group, &u.User); err != nil {
			log.Println(err)
			return
		}

		users = append(users, u)
	}

	return
}

func GetGroupUserList(tx *sql.Tx, aspect, group string) (users []string, err error) {
	var lis []GroupUser

	var group_id int
	group_id, err = GetGroupId(tx, aspect, group, false)

	if lis, err = GetGroupUsers(tx, group_id); err != nil {
		return
	}

	for _, u := range lis {
		users = append(users, u.User)
	}

	return
}

func GetGroupRoles(tx *sql.Tx, group_id int) (roles []GroupRole, err error) {
	var rows *sql.Rows

	rows, err = sq.Select("`group_id`", "`aspect`",  "`role`", "`assign_aspect`", "`assign_group`").
		From("`group_roles`").
		Where(sq.Eq{
			"`group_id`": group_id}).
		RunWith(tx).Query()

	if err != nil {
		log.Println(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var u GroupRole
		if err = rows.Scan(&u.GroupId, &u.Aspect, &u.Role, &u.AssignAspect, &u.AssignGroup); err != nil {
			log.Println(err)

			return
		}

		roles = append(roles, u)
	}

	return
}

func GetGroupRoleList(tx *sql.Tx, aspect, group string) (roles []string, err error) {
	var lis []GroupRole

	var group_id int
	group_id, err = GetGroupId(tx, aspect, group, false)

	if lis, err = GetGroupRoles(tx, group_id); err != nil {
		return
	}

	for _, u := range lis {
		roles = append(roles, u.Aspect + "/" + u.Role)
	}

	return
}
/*
func PutGroupUsers(tx *sql.Tx, aspect, group string, newUsers []string) (err error) {
	var oldUsers []string

	var group_id int
	var ok bool
	if ok, err = HasGroup(tx, aspect, group); err != nil {
		return
	}

	if ok {
		group_id, err = GetGroupId(tx, aspect, group, true)
	} else {
		group_id, err = PutGroupId(tx, aspect, group)
	}
	if err != nil {
		return
	}

	if oldUsers, err = GetGroupUserList(tx, aspect, group); err != nil {
		return
	}

	add := arrayDisjunct(oldUsers, newUsers)
	for _, n := range add {
		if err = PutGroupUser(tx, group_id, n); err != nil {
			return
		}
	}

	del := arrayDisjunct(newUsers, oldUsers)
	for _, n := range del {
		if err = DeleteGroupUser(tx, group_id, n); err != nil {
			return
		}
	}

	return
}
*/


func PutGroupUser(tx *sql.Tx, aspect, group, user string) (ok bool, err error) {

	var group_id int
	if ok, err = HasGroup(tx, aspect, group); err != nil {
		return
	}

	if ok {
		group_id, err = GetGroupId(tx, aspect, group, true)
	} else {
		group_id, err = PutGroupId(tx, aspect, group)
	}
	if err != nil {
		return
	}

	if ok, err = HasGroupUser(tx, group_id, user, true); err != nil {
		return
	}

	if !ok {
		return
	}

	err = PutGroupUserId(tx, group_id, user)

	return
}

func DeleteGroupUser(tx *sql.Tx, aspect, group, user string) (err error) {

	var group_id int
	var ok bool
	if ok, err = HasGroup(tx, aspect, group); err != nil {
		return
	}

	if ok {
		group_id, err = GetGroupId(tx, aspect, group, true)
	} else {
		group_id, err = PutGroupId(tx, aspect, group)
	}
	if err != nil {
		return
	}

	if ok, err = HasGroupUser(tx, group_id, user, true); err != nil {
		return
	}

	if !ok {
		return
	}

	err = DeleteGroupUserId(tx, group_id, user)

	return
}


func PutGroupUserId(tx *sql.Tx, group_id int, user string) (err error) {

	log.Printf("ADD: %d : %s", group_id, user)
	_, err = sq.Insert("`group_user`").
		Columns("`group_id`","`user`").
		Values(group_id, user).
		RunWith(tx).Exec()

	return
}

func DeleteGroupUserId(tx *sql.Tx, group_id int, user string) (err error) {

	log.Printf("DEL: %d : %s", group_id, user)
	_, err = sq.Delete("`group_user`").
		Where(sq.Eq{
			"`group_id`": group_id,
			"`user`": user}).
		RunWith(tx).Exec()

	return
}