package model

import (
	sq "gopkg.in/Masterminds/squirrel.v1"
	"sour.is/x/log"

	"database/sql"
	"fmt"
	"sour.is/x/uuid"
	"strings"
)

var MAX_FILTER int = 40


type PeerNode struct {
	Id      string `json:"peer_id"`
	Name    string `json:"peer_name"`
	Note    string `json:"peer_note,omitempty"`
	Family  int    `json:"peer_family,omitempty"`
	Country string `json:"peer_country,omitempty"`
	Nick    string `json:"peer_nick,omitempty"`
	Owner   string `json:"peer_owner,omitempty"`
	Type    string `json:"peer_type,omitempty"`
	Active  bool   `json:"peer_active,omitempty"`
	Created string `json:"peer_created,omitempty"`
}

func GetPeerList(tx *sql.Tx, owner string) (lis []PeerNode, err error) {
	var rows *sql.Rows

	s := sq.Select("`peer_id`", "`peer_name`").
		From("`peers`.`peers`").
		Where(sq.Eq{"lower(`peer_owner`)": owner})
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

	s := sq.Select("`peer_id`", "`peer_name`", "`peer_note`", "`peer_family`", "`peer_country`", "`peer_nick`", "`peer_owner`", "`peer_type`", "`peer_active`", "`peer_created`").
		From("`peers`.`peers`").
		Where(sq.Eq{"`peer_id`": id})
	if lock {
		s.Suffix("LOCK IN SHARE MODE")
	}

	err = s.RunWith(tx).QueryRow().Scan(&p.Id, &p.Name, &p.Note, &p.Family, &p.Country, &p.Nick, &p.Owner, &p.Type, &p.Active, &p.Created)
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
		Columns("`peer_id`", "`peer_name`", "`peer_note`", "`peer_family`", "`peer_country`", "`peer_nick`", "`peer_owner`", "`peer_type`").
		Values(p.Id, p.Name, p.Note, p.Family, p.Country, p.Nick, p.Owner, p.Type)
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
		Set("`peer_nick`", p.Nick).
		Where(sq.Eq{"`peer_id`": p.Id, "lower(`peer_owner`)": p.Owner}).
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


type RegObject struct {
	Uuid  string        `json:"uuid"`
	Items []RegObjItem `json:"items"`
}
type RegObjItem struct {
	Seq   int    `json:"-"`
	Field string `json:"field"`
	Value string `json:"value"`
}

func GetRegObject(tx *sql.Tx, uuid string) (o RegObject, err error) {
	var rows *sql.Rows

	s := sq.Select("`uuid`", "`seq`", "`field`", "`value`").
		From("`profile`.`reg_values`").
		Where(sq.Eq{"`uuid`": uuid}).OrderBy("`seq`")


	rows, err = s.RunWith(tx).Query()

	if err != nil {
		log.Debug(err)
		return
	}

	defer rows.Close()

	o.Uuid = uuid

	for rows.Next() {
		var i RegObjItem

		if err = rows.Scan(&o.Uuid, &i.Seq, &i.Field, &i.Value); err != nil {
			log.Debug(err)
			return
		}

		o.Items = append(o.Items, i)
	}

	return
}

func PutRegObject(tx *sql.Tx, o RegObject) (err error) {
	log.Debugf("DELETE: %s.%s", o.Uuid)

	s := sq.Delete("`profile`.`reg_values`").Where(sq.Eq{"`uuid`": o.Uuid})
	_, err = s.RunWith(tx).Exec()
	if err != nil {
		log.Debug(err)
		return
	}

	for _, row := range o.Items {
		log.Debugf("ADD: %s : %s = %s", o.Uuid, row.Field, row.Value)

		_, err = sq.Insert("`profile`.`reg_values`").
			Columns("`uuid`", "`field`", "`value`").
			Values(o.Uuid, row.Field, row.Value).
			RunWith(tx).Exec()

		if err != nil {
			log.Debug(err)
			return
		}
	}

	return
}




func GetRegObjects(tx *sql.Tx, query, filter string) (olis []RegObject, err error) {
	var rows *sql.Rows

	log.Info(fmt.Sprintf("Query: %s", query))
	log.Info(fmt.Sprintf("Filter: %s", filter))


	s := sq.Select("`reg_values`.`uuid`", "`reg_values`.`seq`", "`reg_values`.`field`", "`reg_values`.`value`").
		From("`profile`.`reg_values`").OrderBy("`reg_values`.`uuid`","`reg_values`.`seq`")

	for i, o := range simpleParse(query) {
		log.Info(o)
		if i > MAX_FILTER {
			err = fmt.Errorf("Too many filters. [%d]", MAX_FILTER)
			return
		}
		q := sq.Select("`reg_values`.`uuid`").From("`profile`.`reg_values`")

		switch o.Op {
		case "eq":
			q = q.Where(sq.Eq{"`field`": o.Left, "`value`": o.Right})
			break;
		case "neq":
			q = q.Where(sq.And{sq.Eq{"`field`": o.Left}, sq.NotEq{"`value`": o.Right}})
			break;
		case "gt":
			q = q.Where(sq.And{sq.Eq{"`field`": o.Left}, sq.Gt{"`value`": o.Right}})
			break;
		case "lt":
			q = q.Where(sq.And{sq.Eq{"`field`": o.Left}, sq.Lt{"`value`": o.Right}})
			break;
		case "ge":
			q = q.Where(sq.And{sq.Eq{"`field`": o.Left}, sq.GtOrEq{"`value`": o.Right}})
			break;
		case "le":
			q = q.Where(sq.And{sq.Eq{"`field`": o.Left}, sq.LtOrEq{"`value`": o.Right}})
			break;
		case "like":
			q = q.Where("`field` = ? AND `value` LIKE ?", o.Left, o.Right)
			break;
		case "in":
			q = q.Where(sq.Eq{"`field`": o.Left, "`value`": strings.Split(o.Right," ")})
		}
		s = s.JoinClause(q.Prefix("JOIN (").Suffix(fmt.Sprintf(") `r%03d` USING (`uuid`)", i)))
	}

	filter = strings.TrimSpace(filter)
	if filter == "" {
		log.Debug("Return all fields")
	} else if f := strings.Split(filter, ","); len(f) > 0 {
		s = s.Where(sq.Eq{"`reg_values`.`field`": f})
	}

	log.Notice(s.ToSql())

	if query == "" {
		return
	}

	rows, err = s.RunWith(tx).Query()

	if err != nil {
		log.Debug(err)
		return
	}

	defer rows.Close()

	last_uuid := ""
	var curr_uuid string
	var lis []RegObjItem
	var n RegObjItem

	for rows.Next() {
		n = RegObjItem{}

		if err = rows.Scan(&curr_uuid, &n.Seq, &n.Field, &n.Value); err != nil {
			log.Debug(err)
			return
		}

		// log.Info(n)

		if last_uuid != "" && curr_uuid != last_uuid {
			olis = append(olis, RegObject{Uuid: last_uuid, Items: lis})
			lis = []RegObjItem{}
		}

		last_uuid = curr_uuid
		lis = append(lis, n)
	}
	if last_uuid != "" {
		olis = append(olis, RegObject{Uuid: last_uuid, Items: lis})
	}


	return
}

func GetRegAuth(tx *sql.Tx, name string) (o RegObject, err error) {
	var rows *sql.Rows

	s := sq.Select("`mntner`", "`pw_type`", "`pw_value`").
		From("`profile`.`reg_auth`").
		Where(sq.Eq{"`mntner`": name})


	rows, err = s.RunWith(tx).Query()

	if err != nil {
		log.Debug(err)
		return
	}

	defer rows.Close()

	o.Uuid = name

	for rows.Next() {
		var i RegObjItem

		if err = rows.Scan(&o.Uuid, &i.Field, &i.Value); err != nil {
			log.Debug(err)
			return
		}

		log.Debugf("AUTH %s : %s\t%s", o.Uuid, i.Field, i.Value)
		o.Items = append(o.Items, i)
	}

	return
}

func GetParentNetLevel(tx *sql.Tx, min, max, typ string) (level int) {

	wmin := sq.And{sq.Eq{"`field`": "@netmin"}, sq.LtOrEq{"`value`": min}}
	wmax := sq.And{sq.Eq{"`field`": "@netmax"}, sq.GtOrEq{"`value`": max}}

	s := sq.Select("max(if(field='@netlevel',value,''))", "max(if(field='@netmin',value,''))", "max(if(field='@netmax',value,''))").
	        From("`profile`.`reg_values`").
	        Where(sq.Eq{"`field`": []string{"@netlevel", "@netmin", "@netmax"}}).GroupBy("`type`","`name`")

	qmin := sq.Select("`reg_values`.`uuid`").From("`profile`.`reg_values`").Where(wmin)
	s = s.JoinClause(qmin.Prefix("JOIN (").Suffix(") `qmax` USING (`uuid`)"))

	qmax := sq.Select("`reg_values`.`uuid`").From("`profile`.`reg_values`").Where(wmax)
	s = s.JoinClause(qmax.Prefix("JOIN (").Suffix(") `qmin` USING (`uuid`)"))

	qtype := sq.Select("`reg_values`.`uuid`").From("`profile`.`reg_values`").Where(sq.Eq{"`field`": "@type", "`value`": "net"})
	s = s.JoinClause(qtype.Prefix("JOIN (").Suffix(") `qtype` USING (`uuid`)"))


	log.Debug(s.ToSql())

	rows, err := s.RunWith(tx).Query()
	if err != nil {
		log.Debug(err)
		return
	}

	defer rows.Close()

	l := 0
	vmax := ""
	vmin := ""

	log.Debugf("CHILD: %s\t%s\t%s", min, max, typ)

	for rows.Next() {
		if err = rows.Scan(&l, &vmin, &vmax); err != nil {
			log.Debug(err)
			return
		}

		log.Debugf("PARENT: %s\t%s\t%03d", vmin, vmax, l)

		if min == vmin && max == vmax && typ == "net" {
			continue
		}
		if level < l {
			level = l
		}
	}

	log.Debugf("NetLevel: %03d", level)
	return
}

func MoveChildNetLevel(tx *sql.Tx, min, max string, step int) (err error) {

	wmin := sq.And{sq.Eq{"`field`": "@netmin"}, sq.GtOrEq{"`value`": min}}
	wmax := sq.And{sq.Eq{"`field`": "@netmax"}, sq.LtOrEq{"`value`": max}}

	s := sq.Select("`uuid`").
	        Column("lpad(value + ?, 3, '0') as value", step).
		    From("`profile`.`reg_values`").
		    Where(sq.Eq{"`field`": "@netlevel"}).
		    Suffix("FOR UPDATE")

	qmin := sq.Select("`reg_values`.`uuid`").From("`profile`.`reg_values`").Where(wmin)
	s = s.JoinClause(qmin.Prefix("JOIN (").Suffix(") `qmax` USING (`uuid`)"))

	qmax := sq.Select("`reg_values`.`type`", "`reg_values`.`name`").From("`profile`.`reg_values`").Where(wmax)
	s = s.JoinClause(qmax.Prefix("JOIN (").Suffix(") `qmin` USING (`uuid`)"))

	log.Debug(s.ToSql())

	rows, err := s.RunWith(tx).Query()
	if err != nil {
		log.Debug(err)
		return
	}

	defer rows.Close()

	log.Debugf("MOVE BRANCH: %s\t%s\t%s", min, max, step)

	for rows.Next() {
		var n, v string

		if err = rows.Scan(&n, &v); err != nil {
			log.Debug(err)
			return
		}

		log.Debugf("NODE: %s\t%s", n, v)

	}

	return
}



type Ops struct {
	Left  string
	Op    string
	Right string
}

func simpleParse(in string) (out []Ops) {
	log.Info(in)
	items := strings.Split(in, ",")
	for _, i := range items {
		log.Info(i)
		eq := strings.Split(i, "=")
		switch(len(eq)) {
		case 2:
			out = append(out, Ops{eq[0], "eq", eq[1]})
			break
		case 3:
		    if eq[1] == "" {
		    	eq[1] = "eq"
		    }
			out = append(out, Ops{eq[0], eq[1], eq[2]})
		}
	}

	log.Info(out)
	return
}