package route

import (
	"sour.is/x/httpsrv"
	"encoding/json"
	"bytes"
	"sour.is/x/log"
	"encoding/hex"
	"net/http"
	"sour.is/x/ident"
	"fmt"
	"github.com/tumdum/bencoding"
	"golang.org/x/crypto/ed25519"
	"sour.is/x/profile/internal/model"

	"regexp"
	"crypto/rand"
	"sour.is/x/dbm"
	"time"
	"sour.is/x/profile/internal/profile"
	"strings"
)

func init() {
	httpsrv.HttpRegister("auth", httpsrv.HttpRoutes{
		{"putRegister","PUT","/profile/user.register", putRegister},
//		{"getCheckAuth","POST", "/profile/user.checkauth", getCheckAuth},

		{"postSession","POST", "/profile/user.session", postSession},
		{"deleteSession","DELETE", "/profile/user.session", deleteSession},


		// Rubicon Compatibility
		{"rubiconAuthUser", "POST", "/api/v1/authentication/authenticateUser", rubiconAuthUser},
		{"rubiconCheckToken", "GET", "/api/v1/self/getSelf", rubiconCheckToken},
		{"rubiconSignoutToken", "GET", "/api/v1/authentication/signoutToken", rubiconSignoutToken},

		// OAuth Compatibility
		{"oauthAuth", "GET", "/oauth/auth", oauthAuth},
		{"oauthToken", "POST", "/oauth/token", oauthToken},
		{"oauthToken", "GET", "/oauth/user", oauthUser},

	})
}

func oauthAuth(w http.ResponseWriter, r *http.Request) {}
func oauthToken(w http.ResponseWriter, r *http.Request) {}
func oauthUser(w http.ResponseWriter, r *http.Request) {
	o := OAuthUser{}

	writeObject(w, http.StatusOK, o)
}

type OAuthUser struct {
	Id               int     `json:"id"`
	Username         string  `json:"username"`
	Email            string  `json:"email"`
	Name             string  `json:"name"`
	State            string  `json:"state"`
	AvatarUrl        string  `json:"avatar_url"`
	WebUrl           string  `json:"web_url"`
	CreatedAt        string  `json:"created_at"`
	IsAdmin          bool    `json:"is_admin"`
	Bio              string  `json:"bio"`
	Location         string  `json:"location"`
	Skype            string  `json:"skype"`
	LinkedIn         string  `json:"linkedin"`
	Twitter          string  `json:"twitter"`
	WebsiteUrl       string  `json:"website_url"`
	Organization     string  `json:"organization"`
	LastSignInAt     string  `json:"last_sign_in_at"`
	ConfirmedAt      string  `json:"confirmed_at"`
	ColorSchemeId    int     `json:"color_scheme_id"`
	ProjectsLimit    int     `json:"projects_limit"`
	CurrentSignInAt  string  `json:"current_sign_in_at"`
	Identities       []OAuthIdentity  `json:"identities"`
	CanCreateGroup   bool    `json:"can_create_group"`
	CanCreateProject bool    `json:"can_create_project"`
	TwoFactorEnabled bool    `json:"two_factor_enabled"`
	External         bool    `json:"external"`
}
type OAuthIdentity struct {
	Provider string  `json:"provider"`
	ExternId string  `json:"extern_id"`
}

func putRegister(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	tx, err := dbm.GetDB()
	defer checktx(tx, err)

	var reg UserCredential
	if err = json.NewDecoder(r.Body).Decode(&reg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "MALFORMED_REQUEST")
		return
	}

	var ok bool

	if ok, err = regexp.Match(`[a-z0-9\\-\\_]+`, []byte(reg.Ident)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "BAD_USERNAME")
		return
	}

	// Hash the username and password.
	rnd := make([]byte, 32)
	rand.Read(rnd)
	salt := fmt.Sprintf("%x", rnd)
	pkey, err := profile.PassEd25519(reg.Ident, reg.Password, salt)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "UNKNOWN_ERR")
		return
	}

	keys := make(map[string]string)
	keys["ed25519"] = pkey
	keys["salt"] = salt

	atUser := "@" + reg.Ident

	if ok, err = model.HasHash(tx, atUser, "ident"); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if ok {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, "ALREADY_REGISTERED")
		return
	}

	// Write to ident hash.
	if err = model.PutHashMap(tx, atUser, "ident", keys); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = model.PutGroupUser(tx, "default", "USERS", reg.Ident); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "")
}

type UserCredential struct {
	Ident    string `json:"ident"`
	Aspect   string `json:"aspect"`
	Password string `json:"password"`
}
type UserSession struct {
	profile.Profile
	Token   string  `json:"token"`
	Expires int64     `json:"expires"`
}

func postSession(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	var err error
	var cred UserCredential

	if err = json.NewDecoder(r.Body).Decode(&cred); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "MALFORMED_REQUEST")
		return
	}
	if cred.Ident == "" || cred.Password == ""{
		writeMsg(w, http.StatusForbidden, "AUTH_FAILED")
		return
	}

	var ok bool
	if ok, err = profile.CheckPassword(cred.Ident, cred.Password); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR")
		return
	}
	if !ok {
		writeMsg(w, http.StatusForbidden, "AUTH_FAILED")
		return
	}

	var key string
	var expires time.Time
	var p profile.Profile

	if key, expires, p, err = profile.MakeSession(cred.Aspect, cred.Ident, profile.ProfileAll); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR")
		return
	}

	s := UserSession{p, key, expires.UnixNano()}

	writeObject(w, http.StatusCreated, s)
}
func deleteSession(w http.ResponseWriter, r *http.Request) {

	token := r.Header.Get("authorization")

	var ok bool
	var err error

	if !strings.HasPrefix("souris", token) {
		w.Header().Add("www-authenticate", `souris realm="Souris Token"`)
		writeMsg(w, http.StatusUnauthorized, "Authorization Required")
		return
	}

	token = strings.TrimPrefix("souris ", token)

	if ok, err = profile.DeleteSession(token); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR")
		return
	}
	if !ok {
		writeMsg(w, http.StatusForbidden, "NO_SESSION")
		return
	}

	writeObject(w, http.StatusNoContent, "LOGGED_OUT")
}

//!- Rubicon Compatibility

type RubiconCredentials struct {
	Username string
	Password string
	GenerateToken string
}
type RubiconUserInfo struct {
	UserId int64 `json:"userId"`
	UserName string `json:"userName"`
	Email string `json:"email"`
	FirstName string `json:"firstName"`
	LastName string `json:"lastName"`
}
type RubiconUserAuth struct {
	Token string `json:"token"`
	Expires int64 `json:"expires"`
	User RubiconUserInfo `json:"userInfo"`
}

func rubiconAuthUser(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	var err error
	var creds RubiconCredentials

	if err = json.NewDecoder(r.Body).Decode(&creds); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "MALFORMED_REQUEST")
		return
	}

	var ok bool
	if ok, err = profile.CheckPassword(creds.Username, creds.Password); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR")
		return
	}
	if !ok {
		writeMsg(w, http.StatusForbidden, "AUTH_FAILED")
		return
	}

	var key string
	var expires time.Time
	var p profile.Profile

	if key, expires, p, err = profile.MakeSession("rubicon", creds.Username, profile.ProfileGlobal); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR")
		return
	}

	ua := RubiconUserAuth{
		key,
		expires.UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond)),
		RubiconUserInfo{
			1,
			creds.Username,
			p.GlobalMap["email"],
			p.GlobalMap["first_name"],
			p.GlobalMap["last_name"],
		},
	}

	writeObject(w, http.StatusCreated, ua)
}
func rubiconCheckToken(w http.ResponseWriter, r *http.Request) {

	token := r.URL.Query().Get("user_token")

	var ok bool
	var err error
	var p profile.Profile

	if ok, p, err = profile.CheckSession("rubicon", token, profile.ProfileGlobal); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR")
		return
	}
	if !ok {
		writeMsg(w, http.StatusForbidden, "AUTH_FAILED")
		return
	}

	ui := RubiconUserInfo{
		1,
		p.Ident,
		p.GlobalMap["email"],
		p.GlobalMap["first_name"],
		p.GlobalMap["last_name"],
	}

	writeObject(w, http.StatusOK, ui)
}
func rubiconSignoutToken(w http.ResponseWriter, r *http.Request) {

	token := r.URL.Query().Get("user_token")

	var ok bool
	var err error

	if ok, err = profile.DeleteSession(token); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR")
		return
	}
	if !ok {
		writeMsg(w, http.StatusForbidden, "NO_SESSION")
		return
	}

	writeObject(w, http.StatusNoContent, "LOGGED_OUT")
}


// Check Auth

func getCheckAuth(w http.ResponseWriter, r *http.Request) {
/*
	i := ident.Ident{}


	t := AuthToken{
		i.Identity(),
		i.Aspect(),
		time.Now().UnixNano(),
		time.Now().AddDate(0,0,1).UnixNano(),
		[]byte(""),
	}
	s, _ := t.Sign(`4df1520466968debab48fc9711ac68f43e2e5a867669b8f53f5b7e6ed7df0ace`)

	w.WriteHeader(http.StatusOK)
	w.Write(s.Json())
*/
}
func getCheckAuth2(w http.ResponseWriter, r *http.Request, i ident.Ident) {
	s := `{"sig": "TUzn2hDuXy0npeadUUOIa90OTE/oKMH2zr1RWGEWQYNSbrNPlJ9HbZ8cRuihFBHAcBICaVi4lgJkZGk0Fep3CQ==",
    	   "token": {
        		"aspect": "default",
        		"expire": 1490301398018041150,
        		"ident": "@nurtic-vibe",
        		"nonce": 1490214998018041124,
        		"pkey": "KxEK61f8C1QTvUbi4W4X0cyEzykX4BRLeywz45QC4m0="}}`

	var a AuthSignature
	err := json.Unmarshal([]byte(s), &a)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s [%t]", a, a.Verify())
}

type AuthToken struct {
	Ident  string   `json:"ident",bencoding:"ident"`
	Aspect string   `json:"aspect",bencoding:"aspect"`
	Nonce  int64    `json:"nonce",bencoding:"nonce"`
	Expire int64    `json:"expire",bencoding:"expire"`
	PubKey []byte   `json:"pkey",bencoding:"pkey"`
}
type AuthSignature struct {
	Token AuthToken `json:"token"`
	Signature []byte `json:"sig"`
}

func (t AuthToken) Bytes() []byte {
	b, err := bencoding.Marshal(t)
	if err != nil {
		return []byte{}
	}
	return b
}
func (t AuthToken) Json() []byte {
	b, err := json.Marshal(t)
	if err != nil {
		return []byte{}
	}
	return b
}
func (t AuthToken) Sign(skey string) (s AuthSignature, err error) {

	sk, err := dec(skey)
	if err != nil {
		return s, err
	}
	r := bytes.NewReader(sk)
	_, skey_bytes, err := ed25519.GenerateKey(r)
	log.Printf("%x %x", sk, skey_bytes)

	t.PubKey = skey_bytes[32:]

	b := t.Bytes()
	sig := ed25519.Sign(skey_bytes, b[:])

	s = AuthSignature{t, sig}

	return s, nil
}

func (t AuthSignature) String() string { return string(t.Json()) }
func (t AuthSignature) Bytes() []byte {
	b, err := bencoding.Marshal(t)
	if err != nil {
		return []byte{}
	}
	return b
}
func (t AuthSignature) Json() []byte {
	s, err := json.Marshal(t)
	if err != nil {
		return []byte{}
	}
	return s
}
func (t AuthSignature) Verify() bool {
	vfy := ed25519.Verify(t.Token.PubKey, t.Token.Bytes(), t.Signature)
	return vfy
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
