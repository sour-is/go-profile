package profile

import (
	"database/sql"
	"sour.is/x/dbm"
	"sour.is/x/profile/internal/model"
)

func GetHashMap(aspect, name string) (ok bool, hash map[string]string, err error) {
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		hash, ok, err = model.GetHashMap(tx, aspect, name)
		return
	})

	return
}

func PutHashMap(aspect, name string, keys map[string]string) (ok bool, err error) {
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		err = model.PutHashMap(tx, aspect, name, keys)
		return
	})

	return
}

func DeleteHashMap(aspect, name string) (err error) {
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		_, err = model.DeleteHashMap(tx, aspect, name)
		return
	})

	return
}
