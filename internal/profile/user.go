package profile

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"

	"sour.is/x/toolbox/dbm"
	"sour.is/x/toolbox/httpsrv"
	"sour.is/x/toolbox/log"
	"sour.is/x/profile/internal/model"

	"crypto/sha256"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/scrypt"

	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"errors"
	"strconv"
	"strings"
	"time"
)

type Profile struct {
	Ident     string            `json:"ident"`
	Aspect    string            `json:"aspect"`
	GlobalMap map[string]string `json:"site"`
	AspectMap map[string]string `json:"app"`
	LocalMap  map[string]string `json:"local"`
	Roles     []string          `json:"roles"`
	Groups    []string          `json:"groups"`
	LastLogin string            `json:"last_login"`
}

func init() {
	go CronSessions()
}

const (
	ProfileGlobal = 1 << iota
	ProfileAspect
	ProfileLocal
	ProfileOther = ProfileGlobal | ProfileAspect
	ProfileAll   = ProfileGlobal | ProfileAspect | ProfileLocal
	ProfileNone  = 0
)

func getUserProfileTx(tx *sql.Tx, aspect, user string, flag int) (p Profile, err error) {
	user = strings.ToLower(user)
	atUser := "@" + user

	roles := make([]string, 0, 1)
	groups := make([]string, 0, 1)

	global := make(map[string]string)
	app := make(map[string]string)
	local := make(map[string]string)

	var m map[string]string

	if roles, err = model.GetUserRoleList(tx, aspect, user); err != nil {
		return
	}
	if groups, err = model.GetUserGroupList(tx, aspect, user); err != nil {
		return
	}

	if flag&ProfileGlobal != 0 {
		if global, _, err = model.GetHashMap(tx, "global", "default"); err != nil {
			return
		}
		if m, _, err = model.GetHashMap(tx, "global", atUser); err != nil {
			return
		}
		for k, v := range m {
			global[k] = v
		}
		//log.Debugf("Global: %s", global)
	}

	if flag&ProfileAspect != 0 {
		if app, _, err = model.GetHashMap(tx, aspect, "default"); err != nil {
			return
		}
		if m, _, err = model.GetHashMap(tx, aspect, atUser); err != nil {
			return
		}
		for k, v := range m {
			app[k] = v
		}
		//log.Debugf("Aspect: %s", app)
	}

	if flag&ProfileLocal != 0 {
		if local, _, err = model.GetHashMap(tx, atUser, "default"); err != nil {
			return
		}
		if m, _, err = model.GetHashMap(tx, atUser, aspect); err != nil {
			return
		}
		for k, v := range m {
			local[k] = v
		}
		//log.Debugf("User: %s", local)
	}

	var last_log string
	if last_log, _, err = model.GetHashValue(tx, atUser, "ident", "last_login"); err != nil {
		return
	}

	p = Profile{user, aspect, global, app, local, roles, groups, last_log}
	return
}
func putUserProfileTx(tx *sql.Tx, aspect, user string, profile Profile) (err error) {
	atUser := "@" + strings.ToLower(user)
	var defaults map[string]string

	up := make(map[string]string)
	if defaults, _, err = model.GetHashMap(tx, "global", "default"); err != nil {
		return
	}
	for k, v := range profile.GlobalMap {
		var value string
		var present bool

		value, present = defaults[k]
		if present && v == value {
			continue
		}

		up[k] = v
	}
	//log.Debugf("Global: %s", up)
	model.PutHashMap(tx, "global", atUser, up)

	up = make(map[string]string)
	if defaults, _, err = model.GetHashMap(tx, aspect, "default"); err != nil {
		return
	}
	for k, v := range profile.AspectMap {
		var value string
		var present bool

		value, present = defaults[k]
		if present && v == value {
			continue
		}

		up[k] = v
	}
	//log.Debugf("Aspect: %s", up)
	model.PutHashMap(tx, aspect, atUser, up)

	up = make(map[string]string)
	if defaults, _, err = model.GetHashMap(tx, atUser, "default"); err != nil {
		return
	}
	for k, v := range profile.LocalMap {
		var value string
		var present bool

		value, present = defaults[k]
		if present && v == value {
			continue
		}

		up[k] = v
	}
	//log.Debugf("Local: %s", up)
	model.PutHashMap(tx, atUser, aspect, up)

	return
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
func PutUserProfile(aspect, user string, profile Profile) (err error) {

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		err = putUserProfileTx(tx, aspect, user, profile)
		return
	})
	if err != nil {
		return
	}

	return
}

func pow(a, b int64) (p int64) {
	p = 1
	for b > 0 {
		if b&1 != 0 {
			p *= a
		}
		b >>= 1
		a *= a
	}
	return p
}

func passPin(user, password, secret string) (ok bool, check string, err error) {
	h := sha1.New()
	h.Write([]byte(password))
	ck := h.Sum(nil)

	sk, err := dec(secret)
	if err != nil {
		return
	}

	if ck != nil {
		if subtle.ConstantTimeCompare(sk, ck) == 1 {
			ok = true
		}
	}

	check = enc(sk)
	return
}
func passScrypt(user, password, secret, salt string) (ok bool, check string, err error) {
	var N, n, p, r int64
	var sk []byte
	var ck []byte

	if len(secret) == 0 {
		n = 14
		r = 8
		p = 1
		N = pow(2, n)
	} else {
		f := strings.Fields(secret)
		if len(f) < 4 {
			return false, "", errors.New("Invalid Secret")
		}

		n, _ = strconv.ParseInt(f[0], 16, 64)
		r, _ = strconv.ParseInt(f[1], 16, 64)
		p, _ = strconv.ParseInt(f[2], 16, 64)
		N = pow(2, n)

		ck, _ = dec(f[3])
	}

	sk, err = scrypt.Key([]byte(password), []byte(salt), int(N), int(r), int(p), 32)
	if err != nil {
		return
	}

	// Get the pubkey from hash
	read := bytes.NewReader(sk)
	sk, _, err = ed25519.GenerateKey(read)

	if ck != nil {
		if subtle.ConstantTimeCompare(sk, ck) == 1 {
			ok = true
		}
	}

	check = fmt.Sprintf("%x %x %x %s", n, r, p, enc(sk))

	return
}
func passEd25519(user, password, secret, salt string) (ok bool, check string, err error) {
	// Hash the username and password.
	h := sha256.New()
	h.Write([]byte(user))
	h.Write([]byte(password))
	h.Write([]byte(salt))

	var ck []byte
	var sk []byte
	sk = h.Sum(nil)

	// Get the pubkey from hash
	read := bytes.NewReader(sk)
	sk, _, err = ed25519.GenerateKey(read)

	check = enc(sk)

	if len(secret) > 0 {
		if ck, err = dec(secret); err != nil {
			return
		}

		if subtle.ConstantTimeCompare(sk, ck) == 1 {
			ok = true
		}
	}

	return
}

func CreateUser(user, password string) (ok bool, err error) {
	// Hash the username and password.
	rnd := make([]byte, 32)
	rand.Read(rnd)
	salt := fmt.Sprintf("%x", rnd)

	_, ed25519_pass, err := passEd25519(user, password, "", salt)
	if err != nil {
		log.Errorf("CreateUser: %s", err.Error())
		return
	}

	_, scrypt_pass, err := passScrypt(user, password, "", salt)
	if err != nil {
		log.Errorf("CreateUser: %s", err.Error())
		return
	}

	keys := make(map[string]string)

	keys["ed25519"] = ed25519_pass
	keys["scrypt+ed25519"] = scrypt_pass
	keys["salt"] = salt

	atUser := "@" + strings.ToLower(user)

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if ok, err = model.HasHash(tx, atUser, "ident"); err != nil {
			return
		}
		if ok {
			ok = false
			return
		}
		ok = true

		// Write to ident hash.
		if err = model.PutHashMap(tx, atUser, "ident", keys); err != nil {
			return
		}

		// Write to profile hash.
		if err = model.PutHashMap(tx, "global", atUser, map[string]string{"displayName": user}); err != nil {
			return
		}

		if _, err = model.PutGroupUser(tx, "default", "USERS", user); err != nil {
			return
		}

		return
	})

	return

}
func CheckPassword(user, password string) (ok bool, err error) {
	atUser := "@" + strings.ToLower(user)
	var passwd map[string]string

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if ok, err = model.HasHash(tx, atUser, "ident"); err != nil {
			return
		}
		if !ok {
			return
		}
		passwd, ok, err = model.GetHashMap(tx, atUser, "ident")

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
	var secret string
	var present bool

	ok = false
	if secret, present = passwd["scrypt+ed25519"]; present {
		if ok, _, err = passScrypt(user, password, secret, salt); err != nil {
			return
		}
	} else if secret, present := passwd["ed25519"]; present {
		if ok, _, err = passEd25519(user, password, secret, salt); err != nil {
			return
		}

		if ok {
			log.Debug("Adding scrypt password for ", user)

			if _, passwd["scrypt+ed25519"], err = passScrypt(user, password, "", salt); err != nil {
				return
			}
		}
	} else if secret, present := passwd["pin"]; present {
		if ok, _, err = passPin(user, password, secret); err != nil {
			return
		}

		if ok {
			log.Debug("Adding scrypt password for ", user)

			if _, passwd["scrypt+ed25519"], err = passScrypt(user, password, "", salt); err != nil {
				return
			}
		}
	}
	if !ok {
		return
	}

	passwd["last_login"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		// Write to ident hash.
		if err = model.PutHashMap(tx, atUser, "ident", passwd); err != nil {
			return
		}

		return
	})

	return
}
func SetPassword(user, password string) (ok bool, err error) {
	// Hash the username and password.
	rnd := make([]byte, 32)
	rand.Read(rnd)
	salt := fmt.Sprintf("%x", rnd)

	_, ed25519_pass, err := passEd25519(user, password, "", salt)
	if err != nil {
		log.Errorf("CreateUser: %s", err.Error())
		return
	}

	_, scrypt_pass, err := passScrypt(user, password, "", salt)
	if err != nil {
		log.Errorf("CreateUser: %s", err.Error())
		return
	}

	keys := make(map[string]string)
	atUser := "@" + strings.ToLower(user)

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if ok, err = model.HasHash(tx, atUser, "ident"); err != nil {
			return
		}
		if !ok {
			return
		}

		if keys, _, err = model.GetHashMap(tx, atUser, "ident"); err != nil {
			return
		}

		keys["ed25519"] = ed25519_pass
		keys["scrypt+ed25519"] = scrypt_pass
		keys["salt"] = salt

		// Write to ident hash.
		if err = model.PutHashMap(tx, atUser, "ident", keys); err != nil {
			return
		}

		return
	})

	return
}

func dec(src string) (dst []byte, err error) {
	s := []byte(src)
	dst = make([]byte, hex.DecodedLen(len(s)))
	_, err = hex.Decode(dst, s)
	return dst, err
}
func enc(b []byte) string {
	return fmt.Sprintf("%x", b)
}

type Session struct {
	Ident   string `json:"ident"`
	Aspect  string `json:"aspect"`
	Created int64  `json:"created"`
	Expires int64  `json:"expires"`
}

func MakeSession(aspect, user string, flag int) (token string, expires time.Time, p Profile, err error) {
	rnd := make([]byte, 16)
	rand.Read(rnd)
	token = enc(rnd)

	session := make(map[string]string)
	session["ident"] = strings.ToLower(user)
	session["aspect"] = aspect
	session["created"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")

	expires = time.Now().Add(2 * time.Hour)
	session["expires"] = expires.UTC().Format("2006-01-02T15:04:05Z")

	profile_aspect := "default"
	if aspect != "*" {
		profile_aspect = aspect
	}

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if err = model.PutHashMap(tx, "session", token, session); err != nil {
			return
		}
		p, err = getUserProfileTx(tx, profile_aspect, user, flag)

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
		if session, ok, err = model.GetHashMap(tx, "session", token); err != nil {
			return
		}
		if !ok {
			return
		}

		now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		expires := session["expires"]

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
func GetSession(token string) (ok bool, session Session, err error) {
	var m map[string]string

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if ok, err = model.HasHash(tx, "session", token); err != nil {
			return
		}
		if !ok {
			return
		}
		if m, ok, err = model.GetHashMap(tx, "session", token); err != nil {
			return
		}
		if !ok {
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
		ok, err = model.DeleteHashMap(tx, "session", token)

		return
	})

	return
}
func CronSessions() {
	log.Debug("[CRON Session] Waiting for HTTP Startup")
	httpsrv.WaitShutdown.Add(1)
	<-httpsrv.SignalStartup

	log.Debug("[CRON Session] Initialized.")
Exit:
	for {
		select {
		case <-httpsrv.SignalShutdown:
			log.Notice("[CRON Session] Cleaning up.")
			break Exit
		case <-time.After(time.Hour * time.Duration(4)):
			log.Debug("[CRON Session] Clearing expired sessions")
			CleanSessions()
		}
	}

	httpsrv.WaitShutdown.Done()
}
func CleanSessions() {
	var err error
	var lis []model.HashValue
	now := time.Now().UnixNano()

	w := make(map[string]interface{})
	w["aspect"] = "session"
	w["hash_key"] = "expires"

	err = dbm.Transaction(func(tx *sql.Tx) (err error) {
		if lis, err = model.FindHashValue(tx, w); err != nil {
			return
		}
		for _, n := range lis {
			expire, _ := strconv.ParseInt(n.Value, 10, 64)

			if expire < now {
				log.Debugf("[CRON Session] Remove: %s", n.Name)
				err = model.DeleteHashMapId(tx, n.HashId)
			}
			if err != nil {
				return
			}
		}

		return
	})
	if err != nil {
		log.Warning(err.Error())
	}
}
