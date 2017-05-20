package model

import (
	sq "gopkg.in/Masterminds/squirrel.v1"
	"sour.is/x/log"

	"database/sql"
	"fmt"
	"sour.is/x/uuid"
)

type PeerNode struct {
	Id      string `json:"peer_id"`
	Name    string `json:"peer_name"`
	Note    string `json:"peer_note,omitempty"`
	Family  int    `json:"peer_family,omitempty"`
	Country string `json:"peer_country,omitempty"`
	Nick    string `json:"peer_nick,omitempty"`
	Type    string `json:"peer_type,omitempty"`
	Active  bool   `json:"peer_active,omitempty"`
	Created string `json:"peer_created,omitempty"`
}

func GetPeerList(tx *sql.Tx, nick string) (lis []PeerNode, err error) {
	var rows *sql.Rows

	s := sq.Select("`peer_id`", "`peer_name`").
		From("`peers`.`peers`").
		Where(sq.Eq{"lower(`peer_nick`)": nick})
	rows, err = s.RunWith(tx).Query()

	if err != nil {
		log.Debug(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var n PeerNode
		if err = rows.Scan(&n.Id, &n.Name); err != nil {
			log.Debug(err)
			return
		}

		lis = append(lis, n)
	}

	return
}
func HasPeerNode(tx *sql.Tx, id string, lock bool) (_ bool, err error) {

	s := sq.Select("`peer_id`").
		From("`peers`.`peers`").
		Where(sq.Eq{"`peer_id`": id})

	if lock {
		s.Suffix("LOCK IN SHARE MODE")
	}

	var ck string
	err = s.RunWith(tx).QueryRow().Scan(&ck)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		log.Debug(err.Error())
		return
	}

	return id == ck, nil
}
func GetPeerNode(tx *sql.Tx, id string, lock bool) (p PeerNode, ok bool, err error) {

	s := sq.Select("`peer_id`", "`peer_name`", "`peer_note`", "`peer_family`", "`peer_country`", "`peer_nick`", "`peer_type`", "`peer_active`", "`peer_created`").
		From("`peers`.`peers`").
		Where(sq.Eq{"`peer_id`": id})
	if lock {
		s.Suffix("LOCK IN SHARE MODE")
	}

	err = s.RunWith(tx).QueryRow().Scan(&p.Id, &p.Name, &p.Note, &p.Family, &p.Country, &p.Nick, &p.Type, &p.Active, &p.Created)
	if err != nil {
		if err == sql.ErrNoRows {
			return p, false, nil
		}

		log.Debug(err.Error())
		return
	}

	ok = true
	return p, ok, err
}
func (p PeerNode) Insert(tx *sql.Tx) (sp PeerNode, err error) {
	p.Id = uuid.V4()

	var res sql.Result
	s := sq.Insert("`peers`.`peers`").
		Columns("`peer_id`", "`peer_name`", "`peer_note`", "`peer_family`", "`peer_country`", "`peer_nick`", "`peer_type`").
		Values(p.Id, p.Name, p.Note, p.Family, p.Country, p.Nick, p.Type)
	res, err = s.RunWith(tx).Exec()

	if err != nil {
		return
	}

	var num int64
	if num, err = res.RowsAffected(); err != nil {
		return
	}
	if num == 0 {
		err = fmt.Errorf("Insert Failed. %d rows affected.", num)
		return
	}

	sp, _, err = GetPeerNode(tx, p.Id, false)
	return
}

func (p PeerNode) Update(tx *sql.Tx) (sp PeerNode, err error) {
	var res sql.Result
	res, err = sq.Update("`peers`.`peers`").
		Set("`peer_name`", p.Name).
		Set("`peer_note`", p.Note).
		Set("`peer_family`", p.Family).
		Set("`peer_country`", p.Country).
		Set("`peer_type`", p.Type).
		Where(sq.Eq{"`peer_id`": p.Id, "lower(`peer_nick`)": p.Nick}).
		RunWith(tx).Exec()
	if err != nil {
		log.Debug(err.Error())
		return
	}

	var num int64
	if num, err = res.RowsAffected(); err != nil {
		return
	}
	if num == 0 {
		// err = fmt.Errorf("Update Failed. %d rows affected.", num)
		// return
	}

	sp, _, err = GetPeerNode(tx, p.Id, false)
	return
}

func DeletePeerNode(tx *sql.Tx, id string) (err error) {

	s := sq.Delete("`peers`.`peers`").
		Where(sq.Eq{"`peer_id`": id})
	_, err = s.RunWith(tx).Exec()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}

		log.Debug(err.Error())
		return
	}

	return
}
