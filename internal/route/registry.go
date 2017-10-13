package route

import (
	"net/http"
	"github.com/gorilla/mux"
	"sour.is/x/profile/internal/model"
	"sour.is/x/dbm"
	"sour.is/x/log"
	"database/sql"
	"encoding/json"
	"strings"
	"fmt"
	"time"
	"sour.is/x/profile/internal/profile"
	"crypto/subtle"
	"bytes"
	"sour.is/x/httpsrv"
	"strconv"
)

func init() {
	httpsrv.HttpRegister("registry", httpsrv.HttpRoutes{
		{"getRegObject", "GET", "/v1/reg/reg.object({name:[0-9a-zA-Z\\-\\.\\_]+})", getRegObject},
		{"putRegObject", "PUT", "/v1/reg/reg.object({name:[0-9a-zA-Z\\-\\.\\_]+})", putRegObject},
		{"deleteRegObject", "DELETE", "/v1/reg/reg.object({name:[0-9a-zA-Z\\-\\.\\_]+})", deleteRegObject},

		{"getRegObjects", "GET", "/v1/reg/reg.objects", getRegObjects},
		{"postRegAuth", "POST", "/v1/reg/reg.auth", postRegAuth},
	})
}

func getRegObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var err error
	var o model.RegObject

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		o, err = model.GetRegObject(tx, name)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	m := [][]string{}
	for _, n := range o.Items {
		m = append(m, []string{n.Field, n.Value})
	}

	writeObject(w, http.StatusOK, m)
}
func putRegObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["name"]

	var err error
	var lis [][]string
	if err = json.NewDecoder(r.Body).Decode(&lis); err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	var o model.RegObject

	for _, row := range lis {
		if strings.HasPrefix( row[0], "@") {

			continue
		}
		o.Items = append(o.Items, model.RegObjItem{Field: row[0], Value: row[1]})
	}

	var meta map[string]string
	meta, err = parseUri(uuid)
	if err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	o.Uuid = meta["uri"]

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if meta["type"] == "net" || meta["type"] == "route" {
			meta["netlevel"] = fmt.Sprintf("%03d", 1 + model.GetParentNetLevel(tx, meta["netmin"], meta["netmax"], meta["type"]))

			var ok bool
			if ok, err = model.HasRegObject(tx, o.Uuid); err != nil {
				return
			} else {
				if !ok {
					model.MoveChildNetLevel(tx, meta["netmin"], meta["netmax"], 1)
				}
			}
		}

		for name, value := range meta {
			o.Items = append(o.Items, model.RegObjItem{Field: "@" + name, Value: value})
		}
		t := time.Now()
		d := fmt.Sprintf(t.UTC().Format("2006-01-02T15:04:05Z0700"))
		o.Items = append(o.Items, model.RegObjItem{Field: "@updated", Value: d})


		err = model.PutRegObject(tx, o)
		if err != nil {
			return
		}

		o, err = model.GetRegObject(tx, o.Uuid)
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
func deleteRegObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["name"]

	meta, err := parseUri(uuid)
	if err != nil {
		writeMsg(w, http.StatusBadRequest, err.Error())
		return
	}

	o := model.RegObject{}
	o.Uuid = meta["uri"]

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {

		o.Items = append(o.Items, model.RegObjItem{Field: "@uri", Value: o.Uuid})

		t := time.Now()
		d := fmt.Sprintf(t.UTC().Format("2006-01-02T15:04:05Z0700"))
		o.Items = append(o.Items, model.RegObjItem{Field: "@deleted", Value: d})

		err = model.PutRegObject(tx, o)
		if err != nil {
			return
		}

		if meta["type"] == "net" || meta["type"] == "route" {
			model.MoveChildNetLevel(tx, meta["netmin"], meta["netmax"], -1)
		}

		o, err = model.GetRegObject(tx, o.Uuid)
		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeText(w, http.StatusNoContent, "")
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

			space := 0
			for _, n := range o.Items {
				if len(n.Field) > space {
					space = len(n.Field)
				}
			}
			space += 2

			for _, n := range o.Items {

				vals := strings.Split(n.Value, "\n")
				log.Info(vals)

				buf.WriteString(n.Field)
				buf.WriteString(":")
				buf.WriteString(strings.Repeat(" ", space - len(n.Field)))
				buf.WriteString(vals[0])
				buf.WriteString("\n")
					
				if len(vals) > 1 {
					for _, val := range vals[1:] {
						buf.WriteString(strings.Repeat(" ", space + 1))
						buf.WriteString(val)
						buf.WriteString("\n")
					}
				}
			}
		}

		writeText(w, http.StatusOK, buf.String())
	}
}


func reverse(ss []string) ([]string){
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
	return ss
}
func fmtIp4(addr, mask string) string {
	log.Debug(addr)

	addr = addr + "000000000"
	log.Debug(addr)
	addr = addr[24:32]
	log.Debug(addr)

	s := make([]int64,0,4)
	for i := 0; i < len(addr); i+=2 {
		log.Debug(addr[i:i+2])

		if n, err := strconv.ParseInt(addr[i:i+2],16, 64); err == nil {
			s = append(s, n)
		} else {
			return ""
		}
	}

	m, err := strconv.ParseInt(mask, 10, 64)
	if err != nil {
		m = 32
	} else {
		m -= 96
	}
	log.Debug(m)

	if m == 32 || m < 0 {
		return fmt.Sprintf("%d.%d.%d.%d", s[0], s[1], s[2], s[3])
	} else {
		return fmt.Sprintf("%d.%d.%d.%d_%d", s[0], s[1], s[2], s[3], m)
	}
}
func fmtIp6(addr, mask string) string {

	m, err := strconv.ParseInt(mask, 10, 64)
	if err != nil {
		m = 128
	}
	log.Debug(m)

	s := make([]string,0,8)
	for i := 0; i < len(addr); i += 4 {
		seg := addr[i:i+4]
		if len(seg) < 4 {
			seg += "000"
			seg = seg[0:4]
		}
		if i!=0 && seg == "0000" {
			seg = ""
		}
		if i>0 && seg == "" {
			continue
		}
		log.Debug("Segment ",seg)
		s = append(s, seg)
	}

	v6 := strings.Join(s, ":")
	if len(s) < 8 {
		v6 += "::"
	}

	if !(m == 128 || m < 0) {
		v6 += fmt.Sprintf("_%d", m)
	}

	return v6
}
func parseUri(s string) (meta map[string]string, err error) {
	meta = make(map[string]string)

	meta["uri"] = s

	log.Debug(s)
	parts := strings.Split(s, ".")
	log.Debugf("Parts: %d", len(parts))

	switch len(parts) {
	case 1:
		meta["space"] = parts[0]


	case 2:
		meta["type"] = parts[1]
		meta["space"] = parts[0]


	default:
		meta["name"] = parts[2]
		meta["type"] = parts[1]
		meta["space"] = parts[0]

		if parts[1] == "net" {
			meta["netmask"] = ""
			parts := strings.Split(parts[2],"_")
			if len(parts) > 2 {
				meta["sub"] = parts[2]
				meta["netmask"] = parts[1]

			} else {
				meta["sub"] = "block"
			}

			var min, max, mask string
			var m int
			min, max, mask, m, err = inet6range(parts[0] + "_" + parts[1])
			if err != nil {
				return
			}

			meta["netmin"] = min
			meta["netmax"] = max
			meta["netmask"] = mask

			t := meta["sub"]
			if strings.HasPrefix(min, "00000000000000000000ffff") {
				meta["family"] = "ipv4"
				meta["name"] = fmtIp4(min, mask)

			} else {
				meta["family"] = "ipv6"
				meta["name"] = fmtIp6(min, mask)
				switch meta["sub"] {
				case "inetnum":
					meta["sub"] = "inet6num"
				case "route":
					meta["sub"] = "route6"
				}
			}

			o := m/4
			if m%4!=0 {
				o+=1
			}
			if meta["sub"] == "block" {
				meta["uri"] = strings.Join([]string{meta["space"], meta["type"], fmt.Sprintf("%01s", min[:o])},".")
			} else {
				meta["uri"] = strings.Join([]string{meta["space"], meta["type"], fmt.Sprintf("%01s_%d_%s", min[:o], m, t)},".")
			}
		} else if parts[1] == "dns" {
			dns := reverse(parts[2:])
			meta["file"] = fmt.Sprintf("%s/%s", meta["type"], strings.Join(dns, "."))
			meta["name"] = fmt.Sprintf("%s", strings.Join(dns, "."))

		}

	}

	if !(meta["type"] == "" || meta["name"] == "") {
		meta["file"] = meta["type"] + "/" + meta["name"]
	}

	return
}
func expand_ipv6(addr string) (string, error) {
	addr = strings.ToLower(addr)
	log.Debug(addr)
	if strings.Contains(addr, "::") {
		if strings.Count(addr, "::") > 1 {
			return "", fmt.Errorf("invalid ipv6: %s", addr)
		}
		addr = strings.Replace(addr, "::", strings.Repeat(":", 9 - strings.Count(addr, ":")), 1)
	}

	if strings.Count(addr, ":") > 7 {
		return "", fmt.Errorf("invalid ipv6: %s", addr)
	}

	segs := []string{}
	for _, i := range strings.Split(addr, ":") {
		segs = append(segs, lpad(i,"0", 4))
	}

	return strings.Join(segs, ""), nil
}
func lpad(s string,pad string, plength int)string{
	for i:=len(s);i<plength;i++{
		s=pad+s
	}
	return s
}
func rpad(s string,pad string, plength int)string{
	for i:=len(s);i<plength;i++{
		s=s+pad
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
func ip4to6(ip uint64) (string, error) {
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
func inetrange(inet string) (min, max, mask string, m int, err error) {
	pfx := strings.Split(inet, "_")
	m, _ = strconv.Atoi(pfx[1])
	m += 96
	mask = fmt.Sprintf("%03d", m)
	min, err = ip4to6(toNum(pfx[0]))
	if err != nil {
		return
	}

	offset, err := strconv.ParseUint(min[m/4:m/4+1],16, 64)
	if err != nil {
		return
	}

	min = fmt.Sprintf("%s%x%s", min[:m/4], offset & (0xf0>>uint(m%4)), strings.Repeat("0", 31 - m/4))
	max = fmt.Sprintf("%s%x%s", min[:m/4], offset | (0x0f>>uint(m%4)), strings.Repeat("f", 31 - m/4))

	return
}
func inet6range(inet string) (min, max, mask string, m int, err error) {
	pfx := strings.Split(inet, "_")
	m, _ = strconv.Atoi(pfx[1])
	mask = fmt.Sprintf("%03d", m)

	min, err = expand_ipv6(rpad(pfx[0],"0",32))
	if err != nil {
		return
	}

	min += "0"
	offset, err := strconv.ParseUint(min[m/4:m/4+1],16, 64)
	if err != nil {
		return
	}

	min = rpad(fmt.Sprintf("%s%x", min[:m/4], offset & (0xf0>>uint(m%4))),"0",32)
	max = rpad(fmt.Sprintf("%s%x", min[:m/4], offset | (0x0f>>uint(m%4))),"f",32)

	return
}

