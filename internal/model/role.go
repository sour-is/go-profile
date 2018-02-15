package model

import (
	"database/sql"

	sq "gopkg.in/Masterminds/squirrel.v1"
	"sour.is/go/log"
)

type RoleGroup struct {
	GroupId      int
	Aspect       string
	Role         string
	AssignAspect string
	AssignGroup  string
}

func HasRoleGroup(tx *sql.Tx, aspect, role string, group_id int, lock bool) (bool, error) {

	var ok int
	var err error

	s := sq.Select("count(*)").
		From("`group_role`").
		Where(sq.Eq{
			"`aspect`":   aspect,
			"`role`":     role,
			"`group_id`": group_id,
		})

	if lock {
		s.Suffix("FOR UPDATE")
	}

	s.RunWith(tx).QueryRow().Scan(&ok)

	if err != nil {
		log.Warning(err.Error())
		return false, err
	}

	log.Debugf("Count: %d", ok)

	return ok > 0, err
}

func GetRoleGroups(tx *sql.Tx, aspect, role string, lock bool) (groups []RoleGroup, err error) {

	var rows *sql.Rows

	s := sq.Select("`group_id`, `aspect`", "`role`", "`assign_aspect`", "`assign_group`").
		From("`group_roles`").
		Where(sq.Eq{"`aspect`": aspect, "`role`": role})

	if lock {
		s.Suffix("LOCK FOR UPDATE")
	}

	rows, err = s.RunWith(tx).Query()

	if err != nil {
		log.Warning(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var r RoleGroup
		if err = rows.Scan(&r.GroupId, &r.Aspect, &r.Role, &r.AssignAspect, &r.AssignGroup); err != nil {
			log.Warning(err)
			return
		}

		groups = append(groups, r)
	}

	return
}

func GetRoleGroupList(tx *sql.Tx, aspect, role string) (lis []string, err error) {

	var roles []RoleGroup

	if roles, err = GetRoleGroups(tx, aspect, role, false); err != nil {
		return
	}

	for _, r := range roles {
		lis = append(lis, r.AssignAspect+"/"+r.AssignGroup)
	}

	return
}

func PutRoleGroup(tx *sql.Tx, aspect, role, assign, group string) (ok bool, err error) {

	var group_id int
	if ok, err = HasGroup(tx, assign, group); err != nil {
		return false, err
	}

	if ok {
		group_id, err = GetGroupId(tx, assign, group, true)
	} else {
		group_id, err = PutGroupId(tx, assign, group)
	}
	if err != nil {
		log.Warning(err)
		return false, err
	}

	if ok, err = HasRoleGroup(tx, aspect, role, group_id, true); err != nil {
		log.Warning(err)
		return false, err
	}
	if ok {
		return false, err
	}

	err = PutRoleGroupId(tx, aspect, role, group_id)
	return true, err
}

func DeleteRoleGroup(tx *sql.Tx, aspect, role, assign, group string) (err error) {

	var group_id int
	var ok bool
	if ok, err = HasGroup(tx, assign, group); err != nil {
		return
	}

	if ok {
		group_id, err = GetGroupId(tx, assign, group, true)
	} else {
		group_id, err = PutGroupId(tx, assign, group)
	}
	if err != nil {
		return
	}

	if ok, err = HasRoleGroup(tx, aspect, role, group_id, true); err != nil {
		return
	}
	if !ok {
		return
	}

	err = DeleteRoleGroupId(tx, aspect, role, group_id)

	return
}

func PutRoleGroupId(tx *sql.Tx, aspect, role string, group_id int) error {

	log.Debugf("ADD: %s / %s : %d", aspect, role, group_id)

	_, err := sq.Insert("`group_role`").
		Columns("`aspect`", "`role`", "`group_id`").
		Values(aspect, role, group_id).
		RunWith(tx).Exec()

	return err
}

func DeleteRoleGroupId(tx *sql.Tx, aspect, role string, group_id int) error {

	log.Debugf("DEL: %s / %s : %d", aspect, role, group_id)

	_, err := sq.Delete("`group_role`").
		Where(sq.Eq{
			"`aspect`":   aspect,
			"`role`":     role,
			"`group_id`": group_id,
		}).
		RunWith(tx).Exec()

	return err
}

/*
func GetRoleGroupIds(tx *sql.Tx, aspect, role string) (lis []int, err error) {

	var roles []RoleGroup

	if roles, err = GetRoleGroups(tx, aspect, role, true); err != nil {
		return
	}

	for _, r := range roles {
		lis = append(lis, r.GroupId)
	}

	return
}
*/
/*
func PutRoleGroups(tx *sql.Tx, aspect, role string, newGroups []string) (err error) {
	var oldGroups []string

	if oldGroups, err = GetRoleGroupIds(tx, aspect, role); err != nil {
		return
	}

	add := arrayDisjunctInt(oldGroups, newGroups)
	for _, n := range add {
		if sp := strings.Split(n,"/"); len(sp) > 1 {
			if err = PutRoleGroup(tx, aspect, role, sp[0], sp[1]); err != nil {
				log.Debug(err.Error())
				return
			}
		}
	}

	del := arrayDisjunct(newGroups, oldGroups)
	for _, n := range del {
		if sp := strings.Split(n,"/"); len(sp) > 1 {
			if err = DeleteRoleGroup(tx, aspect, role, sp[0], sp[1]); err != nil {
				log.Debug(err.Error())
				return
			}
		}
	}

	return
}
*/
