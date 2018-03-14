package model

import (
	"database/sql"

	sq "gopkg.in/Masterminds/squirrel.v1"
	"sour.is/x/toolbox/log"
)

type HashValue struct {
	HashId int
	Aspect string
	Name   string
	Key    string
	Value  string
}

func HasHash(tx *sql.Tx, aspect, name string) (bool, error) {
	var ok int
	var err error
	//log.Debugf("CHK: %s/%s", aspect, name)

	err = sq.Select("count(*)").
		From("hash").
		Where(sq.Eq{
			"`aspect`":    aspect,
			"`hash_name`": name}).
		RunWith(tx).QueryRow().Scan(&ok)

	if err != nil {
		log.Warning(err)
		return false, err
	}

	return ok > 0, err
}

func GetHashList(tx *sql.Tx, aspect string) (lis []string, err error) {
	var rows *sql.Rows

	rows, err = sq.Select("`hash_name`").
		From("`hash`").
		Where(sq.Eq{"`aspect`": aspect}).
		RunWith(tx).Query()

	if err != nil {
		log.Warning(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var h string
		rows.Scan(&h)
		lis = append(lis, h)
	}

	return
}

func GetHashId(tx *sql.Tx, aspect, name string, lock bool) (int, error) {
	var id int
	var err error

	s := sq.Select("`hash_id`").
		From("`hash`").
		Where(sq.Eq{
			"`aspect`":    aspect,
			"`hash_name`": name})

	if lock {
		s.Suffix("LOCK IN SHARE MODE")
	}

	err = s.RunWith(tx).QueryRow().Scan(&id)

	if err != nil {
		log.Warning(err.Error())
		return 0, err
	}

	return id, err
}
func PutHashId(tx *sql.Tx, aspect, name string) (int, error) {
	var id int64
	var err error

	var res sql.Result
	res, err = sq.Insert("`hash`").
		Columns("`aspect`", "`hash_name`").
		Values(aspect, name).
		RunWith(tx).Exec()

	if err != nil {
		log.Warning(err.Error())
		return 0, err
	}

	id, err = res.LastInsertId()

	return int(id), err
}
func DeleteHashMapId(tx *sql.Tx, hash_id int) (err error) {

	_, err = sq.Delete("`hash_value`").
		Where(sq.Eq{"`hash_id`": hash_id}).
		RunWith(tx).Exec()

	if err != nil {
		log.Warning(err.Error())
		return
	}

	_, err = sq.Delete("`hash`").
		Where(sq.Eq{"`hash_id`": hash_id}).
		RunWith(tx).Exec()

	if err != nil {
		log.Warning(err.Error())
		return
	}

	return
}

func GetHashMap(tx *sql.Tx, aspect, name string) (hash map[string]string, ok bool, err error) {

	hash = make(map[string]string)
	var hash_id int

	if ok, err = HasHash(tx, aspect, name); err != nil {
		return
	}
	if !ok {
		return
	}

	if hash_id, err = GetHashId(tx, aspect, name, false); err != nil {
		return
	}

	hash, err = GetHashMapId(tx, hash_id)
	//log.Debugf("MAP: %s/%s: %v ? %t", aspect, name, hash, ok)
	return
}
func GetHashMapId(tx *sql.Tx, hash_id int) (hash map[string]string, err error) {

	hash = make(map[string]string)

	var lis []HashValue

	if lis, err = GetHashValues(tx, hash_id); err != nil {
		return
	}

	for _, h := range lis {
		hash[h.Key] = h.Value
	}

	return
}
func PutHashMap(tx *sql.Tx, aspect, hash string, keys map[string]string) (err error) {

	var ok bool
	var hash_id int
	old := make(map[string]string)

	if ok, err = HasHash(tx, aspect, hash); err != nil {
		return
	}
	if ok {
		if hash_id, err = GetHashId(tx, aspect, hash, true); err != nil {
			return
		}
		if old, err = GetHashMapId(tx, hash_id); err != nil {
			return
		}
	} else {
		if hash_id, err = PutHashId(tx, aspect, hash); err != nil {
			return
		}
	}

	oldKeys := arrayKeys(old)
	newKeys := arrayKeys(keys)

	add := arrayDisjunct(oldKeys, newKeys)
	for _, n := range add {
		if err = PutHashValueId(tx, hash_id, n, keys[n]); err != nil {
			return
		}
	}

	chk := arrayIntersect(newKeys, oldKeys)
	for _, n := range chk {
		if keys[n] != old[n] {
			if err = DeleteHashValueId(tx, hash_id, n); err != nil {
				return
			}
			if err = PutHashValueId(tx, hash_id, n, keys[n]); err != nil {
				return
			}
		}
	}

	del := arrayDisjunct(newKeys, oldKeys)
	for _, n := range del {
		if err = DeleteHashValueId(tx, hash_id, n); err != nil {
			return
		}
	}

	if len(add)+len(chk) == 0 {

	}

	return
}
func DeleteHashMap(tx *sql.Tx, aspect, name string) (ok bool, err error) {

	var hash_id int

	if ok, err = HasHash(tx, aspect, name); err != nil {
		return
	}
	if !ok {
		return
	}

	if hash_id, err = GetHashId(tx, aspect, name, false); err != nil {
		return
	}

	err = DeleteHashMapId(tx, hash_id)
	//log.Debugf("MAP: %s/%s: %v ? %t", aspect, name, hash, ok)
	return
}

func GetHashKeyList(tx *sql.Tx, hash_id int) (lis []string, err error) {
	var rows *sql.Rows

	rows, err = sq.Select("`hash_key`").
		From("`hash_value`").
		Where(sq.Eq{
			"`hash_id`": hash_id}).
		RunWith(tx).Query()

	if err != nil {
		log.Warning(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var h string
		if err = rows.Scan(&h); err != nil {
			return
		}

		lis = append(lis, h)
	}

	return
}
func GetHashValues(tx *sql.Tx, hash_id int) (lis []HashValue, err error) {
	var rows *sql.Rows

	rows, err = sq.Select("`hash_id`", "`aspect`", "`hash_name`", "`hash_key`", "`hash_value`").
		From("`hash_values`").
		Where(sq.Eq{"`hash_id`": hash_id}).
		RunWith(tx).Query()

	if err != nil {
		log.Warning(err)
		return lis, err
	}

	defer rows.Close()

	for rows.Next() {
		var h HashValue
		if err = rows.Scan(&h.HashId, &h.Aspect, &h.Name, &h.Key, &h.Value); err != nil {
			return
		}

		lis = append(lis, h)
	}

	return lis, err
}

func HasHashValueId(tx *sql.Tx, hash_id int, key string, lock bool) (bool, error) {
	var ok int
	var err error

	s := sq.Select("count(*)").
		From("`hash_value`").
		Where(sq.Eq{
			"`hash_id`":  hash_id,
			"`hash_key`": key})

	if lock {
		s.Suffix("FOR UPDATE")
	}

	s.RunWith(tx).QueryRow().Scan(&ok)

	if err != nil {
		log.Warning(err.Error())
		return false, err
	}

	return ok > 0, err
}
func GetHashValueId(tx *sql.Tx, hash_id int, key string) (value string, err error) {
	s := sq.Select("`hash_value`").
		From("`hash_value`").
		Where(sq.Eq{
			"`hash_id`":  hash_id,
			"`hash_key`": key})
	s.RunWith(tx).QueryRow().Scan(&value)
	if err != nil {
		return
	}

	return
}
func PutHashValueId(tx *sql.Tx, hash_id int, key, value string) (err error) {

	log.Debugf("ADD: %d : %s = %s", hash_id, key, value)
	_, err = sq.Insert("`hash_value`").
		Columns("`hash_id`", "`hash_key`", "`hash_value`").
		Values(hash_id, key, value).
		RunWith(tx).Exec()

	return
}
func DeleteHashValueId(tx *sql.Tx, hash_id int, key string) (err error) {

	log.Debugf("DEL: %d : %s ", hash_id, key)
	_, err = sq.Delete("hash_value").
		Where(sq.Eq{
			"`hash_id`":  hash_id,
			"`hash_key`": key}).
		RunWith(tx).Exec()

	return
}

func GetHashValue(tx *sql.Tx, aspect, name, key string) (value string, ok bool, err error) {

	var hash_id int
	if ok, err = HasHash(tx, aspect, name); err != nil {
		return
	}
	if !ok {
		return
	}

	hash_id, err = GetHashId(tx, aspect, name, false)
	if err != nil {
		return
	}
	log.Debug("HASH: ", hash_id)

	if ok, err = HasHashValueId(tx, hash_id, key, true); err != nil {
		return
	}
	if !ok {
		return
	}

	value, err = GetHashValueId(tx, hash_id, key)
	log.Debug("VALUE: ", value)

	return
}
func PutHashValue(tx *sql.Tx, aspect, name, key, value string) (err error) {

	var ok bool
	var hash_id int

	if ok, err = HasHash(tx, aspect, name); err != nil {
		return
	}

	if ok {
		hash_id, err = GetHashId(tx, aspect, name, true)
	} else {
		hash_id, err = PutHashId(tx, aspect, name)
	}
	if err != nil {
		return
	}

	if ok, err = HasHashValueId(tx, hash_id, key, true); err != nil {
		return
	}
	if ok {
		if err = DeleteHashValueId(tx, hash_id, key); err != nil {
			return
		}
	}

	err = PutHashValueId(tx, hash_id, name, value)
	return
}
func DeleteHashValue(tx *sql.Tx, aspect, name, key string) (err error) {
	var hash_id int
	var ok bool
	if ok, err = HasHash(tx, aspect, name); err != nil {
		return
	}

	if ok {
		hash_id, err = GetHashId(tx, aspect, name, true)
	} else {
		hash_id, err = PutHashId(tx, aspect, name)
	}
	if err != nil {
		return
	}

	if ok, err = HasHashValueId(tx, hash_id, key, true); err != nil {
		return
	}
	if !ok {
		return
	}

	err = DeleteHashValueId(tx, hash_id, name)
	return
}

func FindHashValue(tx *sql.Tx, where map[string]interface{}) (lis []HashValue, err error) {

	var rows *sql.Rows

	rows, err = sq.Select("`hash_id`", "`aspect`", "`hash_name`", "`hash_key`", "`hash_value`").
		From("`hash_values`").
		Where(sq.Eq(where)).
		RunWith(tx).Query()

	if err != nil {
		log.Warning(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var h HashValue
		if err = rows.Scan(&h.HashId, &h.Aspect, &h.Name, &h.Key, &h.Value); err != nil {
			return
		}

		lis = append(lis, h)
	}

	return
}
