package route

import (
	"database/sql"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"sour.is/x/dbm"
	"sour.is/x/httpsrv"
	"sour.is/x/ident"
//	"sour.is/x/log"
	"sour.is/x/profile/internal/model"
	"strings"
	"bytes"
	"crypto/subtle"
	"sour.is/x/profile/internal/profile"
	"fmt"
	"strconv"
)

func init() {
	httpsrv.IdentRegister("peer", httpsrv.IdentRoutes{
		{"getNodes", "GET", "/v1/peers/peer.nodes", getNodes},
		{"postNode", "POST", "/v1/peers/peer.nodes", putNode},
		{"getNode", "GET", "/v1/peers/peer.node({id})", getNode},
		{"putNode", "PUT", "/v1/peers/peer.node({id})", putNode},
		{"putNode", "DELETE", "/v1/peers/peer.node({id})", deleteNode},
    })

    httpsrv.HttpRegister("registry", httpsrv.HttpRoutes{
		{"getRegObject", "GET", "/v1/reg/reg.object({type},{name:[0-9a-zA-Z\\-\\.:_]+})", getRegObject},
		{"putRegObject", "PUT", "/v1/reg/reg.object({type},{name:[0-9a-zA-Z\\-\\.:_]+})", putRegObject},
		{"getRegObjects", "GET", "/v1/reg/reg.objects", getRegObjects},
		{"postRegAuth", "POST", "/v1/reg/reg.auth", postRegAuth},
	})
}

func getNodes(w http.ResponseWriter, _ *http.Request, i ident.Ident) {
	var lis []model.PeerNode

	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}
	err := dbm.Transaction(func(tx *sql.Tx) (err error) {
		lis, err = model.GetPeerList(tx, i.Identity())
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeObject(w, http.StatusOK, lis)
}
func getNode(w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	id := vars["id"]

	defer r.Body.Close()

	var ok bool
	var err error
	var node model.PeerNode

	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		node, ok, err = model.GetPeerNode(tx, id, false)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeMsg(w, http.StatusNotFound, "Not Found")
		return
	}

	writeObject(w, http.StatusOK, node)
}
func putNode(w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	id := vars["id"]

	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	var err error
	var node model.PeerNode
	if err = json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	node.Id = id
	if node.Owner == "" {
		node.Owner = strings.ToLower(i.Identity())
	}

	if strings.ToLower(node.Owner) != strings.ToLower(i.Identity()) {
		writeMsg(w, http.StatusForbidden, "peer_owner should match user ident "+node.Owner+" == "+i.Identity())
		return
	}

	var ok bool
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		var check model.PeerNode
		if check, ok, err = model.GetPeerNode(tx, id, true); err != nil {
			return
		}
		if !ok {
			ok = true
			node, err = node.Insert(tx)

			return
		}
		if strings.ToLower(check.Owner) != strings.ToLower(i.Identity()) {
			ok = false
			writeMsg(w, http.StatusForbidden, "peer_owner should match user ident "+node.Owner+" == "+i.Identity())
			return
		}
		if node, err = node.Update(tx); err != nil {
			return
		}

		return
	})
	if err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}
	if !ok {
		return
	}

	writeObject(w, http.StatusCreated, node)
}
func deleteNode(w http.ResponseWriter, r *http.Request, i ident.Ident) {
	vars := mux.Vars(r)
	id := vars["id"]

	if !i.LoggedIn() {
		writeMsg(w, http.StatusForbidden, "Access Denied")
		return
	}

	var ok bool
	var err error
	var node model.PeerNode

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		node, ok, err = model.GetPeerNode(tx, id, true)
		if !ok {
			writeMsg(w, http.StatusNotFound, "Not Found")
			return
		}
		if strings.ToLower(node.Owner) != i.Identity() {
			ok = false
			writeMsg(w, http.StatusForbidden, "peer_owner should match user ident: "+node.Owner+" == "+i.Identity())
			return
		}

		ok = true
		model.DeletePeerNode(tx, id)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}
	if !ok {
		return
	}

	writeObject(w, http.StatusNoContent, node)
}

func getRegObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId := vars["type"]
	name := vars["name"]

	var err error
	var o model.RegObject

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		o, err = model.GetRegObject(tx, typeId, name)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	var m [][]string
	for _, n := range o.Items {
		m = append(m, []string{n.Field, n.Value})
	}

	writeObject(w, http.StatusOK, m)
}

func putRegObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	objType := vars["type"]
	name := vars["name"]

	var err error
	var lis [][]string
	if err = json.NewDecoder(r.Body).Decode(&lis); err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	var o model.RegObject
	o.Type = objType
	o.Name = name

	meta := make(map[string]string)

	for _, row := range lis {
		if strings.HasPrefix( row[0], "@") {
			meta[row[0]] = row[1]

			continue
		}
		o.Items = append(o.Items, model.RegObjItem{Field: row[0], Value: row[1]})
	}

	meta["@type"] = objType
	meta["@name"] = name

	switch objType {
	case "inetnum":
		meta["@family"] = "ipv4"
		meta["@type"] = "net"
		min, max, mask, m := inetrange(name)

		meta["@netmin"] = min
		meta["@netmax"] = max
		meta["@netmask"] = mask

		o := m/4
		if m%4!=0 {
			o += 1
		}
		meta["@uri"] = strings.Join([]string{"dn42", "net", fmt.Sprintf("%s_%s_inetnum", min[:o], mask)},".")

		break

	case "route":
		meta["@family"] = "ipv4"
		meta["@type"] = "route"
		min, max, mask, m := inetrange(name)

		meta["@netmin"] = min    // "00000000000000000000ffff00000000"
		meta["@netmax"] = max    // "00000000000000000000ffffffffffff"
		meta["@netmask"] = mask  // "096"

		o := m/4
		if m%4!=0 {
			o += 1
		}
		meta["@uri"] = strings.Join([]string{"dn42", "net", fmt.Sprintf("%s_%s_route", min[:o], mask)},".")

		break

	case "inet6num":
		meta["@family"] = "ipv6"
		meta["@type"] = "net"
		min, max, mask, m := inet6range(name)

		meta["@netmin"] = min
		meta["@netmax"] = max
		meta["@netmask"] = mask

		o := m/4
		if m%4!=0 {
			o += 1
		}
		meta["@uri"] = strings.Join([]string{"dn42", "net", fmt.Sprintf("%s_%s_inetnum", min[:o], mask)},".")

		break

	case "route6":
		meta["@family"] = "ipv6"
		meta["@type"] = "route"

		min, max, mask, m := inet6range(name)

		meta["@netmin"] = min
		meta["@netmax"] = max
		meta["@netmask"] = mask

		o := m/4
		if m%4!=0 {
			o += 1
		}
		meta["@uri"] = strings.Join([]string{"dn42", "net", fmt.Sprintf("%s_%s_route", min[:o], mask)},".")

		break

	case "dns":
		parts := strings.Split(name, ".")
		reverse(parts)
		meta["@uri"] = strings.Join([]string{"dn42", objType, strings.Join(parts, ".")},".")

		break

	default:
		meta["@uri"] = strings.Join([]string{"dn42", objType, name},".")
	}

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if meta["@type"] == "net" || meta["@type"] == "route" {
			meta["@netlevel"] = fmt.Sprintf("%03d", 1 + model.GetParentNetLevel(tx, meta["@netmin"], meta["@netmax"], meta["@type"]))
		}

		for name, value := range meta {
			o.Items = append(o.Items, model.RegObjItem{Field: name, Value: value})
		}

		err = model.PutRegObject(tx, o)
		if err != nil {
			return
		}

		o, err = model.GetRegObject(tx, objType, name)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	var m [][]string
	for _, n := range o.Items {
		m = append(m, []string{n.Field, n.Value})
	}

	writeObject(w, http.StatusOK, m)
}


func postRegAuth(w http.ResponseWriter, r *http.Request) {

	var err error
    var creds profile.RegAuthCreds
	if err = json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	var o model.RegObject
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		o, err = model.GetRegAuth(tx, creds.Username)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	ok := false
	_, check := profile.PassSiska(creds.Username, creds.Password, "")
	for _, n := range o.Items {
		if subtle.ConstantTimeCompare([]byte(check), []byte(n.Value)) == 1 {
			ok = true
		}
	}

    if ok {
		writeText(w, http.StatusOK, "OK")
    } else {
		writeText(w, http.StatusForbidden, "Err")
	}
}


func getRegObjects(w http.ResponseWriter, r *http.Request) {

	var err error
	var lis []model.RegObject

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		lis, err = model.GetRegObjects(tx, r.URL.Query().Get("filter"), r.URL.Query().Get("fields"))
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}



	if strings.Contains(r.Header.Get("accept"),"application/json") {
		m := make([][][]string,0,len(lis))
		for _, o := range lis {
			var l [][]string
			for _, n := range o.Items {
				l = append(l, []string{n.Field, n.Value})
			}
			m = append(m, l)
		}

		writeObject(w, http.StatusOK, m)
	} else {
		buf := new(bytes.Buffer)
		for _, o := range lis {
			buf.WriteString("---\n")
			for _, n := range o.Items {
				buf.WriteString(n.Field)
				buf.WriteString(": ")
				buf.WriteString(n.Value)
				buf.WriteString("\n")
			}
		}

		writeText(w, http.StatusOK, buf.String())
	}
}

func reverse(ss []string) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}
func expand_ipv6(addr string) string {
	addr = strings.ToLower(addr)

	if strings.Contains(addr, "::") {
		if strings.Count(addr, "::") > 1 {
			return ""
		}
		addr = strings.Replace(addr, "::", strings.Repeat(":", 9 - strings.Count(addr, ":")), 1)
	}

	if strings.Count(addr, ":") > 7 {
		return ""
	}

	segs := []string{}
	for _, i := range strings.Split(addr, ":") {
		segs = append(segs, lpad(i,"0", 4))
	}

	return strings.Join(segs, "")
}
func lpad(s string,pad string, plength int)string{
	for i:=len(s);i<plength;i++{
		s=pad+s
	}
	return s
}
func toNum(addr string) (ip uint64) {
	for i, s := range strings.SplitN(addr, ".", 4) {
		n, _ := strconv.Atoi(s)
		ip += uint64(n) * ipow(256, uint64(3 - i))
	}

	return
}
func ip4to6(ip uint64) string {
	return expand_ipv6(fmt.Sprintf("::ffff:%04x:%04x", ip >> 16, ip & 0xffff))
}
func ipow(a, b uint64) uint64 {
	var result uint64 = 1

	for 0 != b {
		if 0 != (b & 1) {
			result *= a
		}
		b >>= 1
		a *= a
	}

	return result
}
func inetrange(inet string) (min, max, mask string, m int) {
	pfx := strings.Split(inet, "_")
	m, _ = strconv.Atoi(pfx[1])
    m += 96
	mask = fmt.Sprintf("%03d", m)


//	n := toNum(pfx[0]) & uint64(0xFFFFFFFF << 32 - m)
	min = ip4to6(toNum(pfx[0]))

//	x := n | uint64(0xFFFFFFFF >> 0 + m)
//	max = ip4to6(x)
	offset, _ := strconv.ParseUint(min[m/4:m/4+1],16, 64)

	min = fmt.Sprintf("%s%x%s", min[:m/4], offset & (0xf0>>uint(m%4)), strings.Repeat("0", 31 - m/4))
	max = fmt.Sprintf("%s%x%s", min[:m/4], offset | (0x0f>>uint(m%4)), strings.Repeat("f", 31 - m/4))

	return
}
func inet6range(inet string) (min, max, mask string, m int) {
	pfx := strings.Split(inet, "_")
	m, _ = strconv.Atoi(pfx[1])
	mask = fmt.Sprintf("%03d", 0 + m)

	min = expand_ipv6(pfx[0])
	offset, _ := strconv.ParseUint(min[m/4:m/4+1],16, 64)

	min = fmt.Sprintf("%s%x%s", min[:m/4], offset & (0xf0>>uint(m%4)), strings.Repeat("0", 31 - m/4))
	max = fmt.Sprintf("%s%x%s", min[:m/4], offset | (0x0f>>uint(m%4)), strings.Repeat("f", 31 - m/4))

	return
}















