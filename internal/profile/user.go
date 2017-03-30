package profile

import (
	"database/sql"
	"sour.is/x/dbm"
	"sour.is/x/profile/internal/model"
	"bytes"
	"encoding/hex"
	"fmt"
	"crypto/sha256"
	"golang.org/x/crypto/ed25519"
	"time"
	"strconv"
	"crypto/rand"
	"sour.is/x/log"
	"crypto/subtle"
)

type Profile struct {
	Ident       string            `json:"ident"`
	Aspect      string            `json:"aspect"`
	GlobalMap   map[string]string `json:"site"`
	AspectMap   map[string]string `json:"app"`
	UserMap     map[string]string  `json:"user"`
	Roles       []string          `json:"roles"`
}

func GetUserProfile(aspect, user string, flag int) (p Profile, err error) {
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		p, err = getUserProfileTx(tx, aspect, user, flag)
		return
	})
	if err != nil {
		return
	}

	return
}

const (
	ProfileGlobal = 1 << iota
	ProfileAspect
	ProfileLocal
	ProfileOther = ProfileGlobal | ProfileAspect
	ProfileAll = ProfileGlobal | ProfileAspect | ProfileLocal
	ProfileNone = 0
)


func getUserProfileTx(tx *sql.Tx, aspect, user string, flag int) (p Profile, err error) {
	atUser := "@" + user

	var roles []string

	global := make(map[string]string)
	app := make(map[string]string)
	local := make(map[string]string)

	var m map[string]string

	if roles, err = model.GetUserRoleList(tx, aspect, user); err != nil {
		return
	}

	if flag & ProfileGlobal != 0 {
		if global, err = model.GetHashMap(tx, "global", "default"); err != nil {
			return
		}
		if m, err = model.GetHashMap(tx, "global", atUser); err != nil {
			return
		}
		for k, v := range m {
			global[k] = v
		}
		log.Debugf("Global: %s", global)
	}

	if flag & ProfileAspect != 0{
		if app, err = model.GetHashMap(tx, aspect, "default"); err != nil {
			return
		}
		if m, err = model.GetHashMap(tx, aspect, atUser); err != nil {
			return
		}
		for k, v := range m {
			app[k] = v
		}
		log.Debugf("Aspect: %s", app)
	}

	if flag & ProfileLocal != 0{
		if local, err = model.GetHashMap(tx, atUser, "default"); err != nil {
			return
		}
		if m, err = model.GetHashMap(tx, atUser, aspect); err != nil {
			return
		}
		for k, v := range m {
			local[k] = v
		}
		log.Debugf("User: %s", local)
	}

	p = Profile{user, aspect, global, app, local, roles}
	return
}



func PassEd25519(user, password, salt string) (string, error) {
	// Hash the username and password.
	h := sha256.New()
	h.Write([]byte(user))
	h.Write([]byte(password))
	h.Write([]byte(salt))
	sk := h.Sum(nil)

	// Get the pubkey from hash
	read := bytes.NewReader(sk)
	pkey, _, err := ed25519.GenerateKey(read)

	return enc(pkey), err
}
func CheckPassword(user, password string) (ok bool, err error) {
	atUser := "@" + user
	var passwd map[string]string

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if ok, err = model.HasHash(tx, atUser, "ident"); err != nil {
			return
		}
		if !ok {
			return
		}
		passwd, err = model.GetHashMap(tx, atUser, "ident")

		return
	})
	if err != nil {
		return false, err
	}
	if !ok {
		return
	}

	// Hash the username and password.
	salt := passwd["salt"]
	pkey := passwd["ed25519"]

	var check string
	if check, err = PassEd25519(user, password, salt); err != nil {
		return
	}

	if subtle.ConstantTimeCompare([]byte(pkey), []byte(check)) == 0 {
		return false, nil
	}
	return true, nil
}
func dec(s string) (b []byte, err error) {
	src := []byte(s)
	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err = hex.Decode(dst, src)
	return dst, err
}
func enc(b []byte) string {
	return fmt.Sprintf("%x", b)
}

func MakeSession(aspect, user string, flag int) (token string, expires time.Time, p Profile, err error) {
	rnd := make([]byte, 16)
	rand.Read(rnd)
	token = enc(rnd)

	session := make(map[string]string)
	session["ident"] = user
	session["aspect"] = aspect
	session["created"] = strconv.FormatInt(time.Now().UnixNano(), 10)

	expires = time.Now().Add(2 * time.Hour)
	session["expires"] = strconv.FormatInt(expires.UnixNano(), 10)

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if err = model.PutHashMap(tx, "session", token, session); err != nil {
			return
		}
		p, err = getUserProfileTx(tx, aspect, user, flag)

		return
	})

	return
}

func CheckSession(aspect, token string, flag int) (ok bool, user Profile, err error) {

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		var session map[string]string
		if ok, err = model.HasHash(tx, "session", token); err != nil {
			return
		}
		if !ok {
			return
		}
		if session, err = model.GetHashMap(tx, "session", token); err != nil {
			return
		}

		now := time.Now().UnixNano()
		expires, _ := strconv.ParseInt(session["expires"], 10, 64)

		ok = now < expires
		if !ok {
			return
		}

		ident := session["ident"]

		if session["aspect"] == "" {
			// Null aspect use provided.
		} else if session["aspect"] == "*" {
			// Star aspect accept provided.
		} else {
			// Set aspect restrict value.
			aspect = session["aspect"]
		}

		user, err = getUserProfileTx(tx, aspect, ident, flag)

		return
	})

	return
}
type Session struct {
	Ident   string  `json:"ident"`
	Aspect  string  `json:"aspect"`
	Created int64   `json:"created"`
	Expires int64   `json:"expires"`
}

func GetSession(token string) (ok bool, session Session, err error) {
	var m map[string]string

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if ok, err = model.HasHash(tx, "session", token); err != nil {
			return
		}
		if !ok {
			return
		}
		if m, err = model.GetHashMap(tx, "session", token); err != nil {
			return
		}

		session.Ident = m["ident"]
		session.Aspect = m["aspect"]
		session.Created, _ = strconv.ParseInt(m["created"], 10, 64)
		session.Expires, _ = strconv.ParseInt(m["expires"], 10, 64)

		return
	})

	return
}


func DeleteSession(token string) (ok bool, err error) {
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {

		if ok, err = model.HasHash(tx, "session", token); err != nil {
			return
		}
		if !ok {
			return
		}

		empty := make(map[string]string)
		if err = model.PutHashMap(tx, "session", token, empty); err != nil {
			return
		}

		return
	})

	return
}