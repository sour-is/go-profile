package route

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dchest/captcha"
	"github.com/gorilla/mux"
	"sour.is/x/toolbox/dbm"
	"sour.is/x/toolbox/httpsrv"
	"sour.is/x/toolbox/ident"
	"sour.is/x/toolbox/log"
	"sour.is/x/profile/internal/model"
	"sour.is/x/profile/internal/profile"
)

func init() {
	httpsrv.HttpRegister("auth", httpsrv.HttpRoutes{
		{"putRegister", "PUT", "/v1/profile/user.register", putRegister},
		{"postSession", "POST", "/v1/profile/user.session", postSession},
		{"deleteSession", "DELETE", "/v1/profile/user.session", deleteSession},

		// Rubicon Compatibility
		{"rubiconAuthUser", "POST", "/api/v1/authentication/authenticateUser", rubiconAuthUser},
		{"rubiconCheckToken", "GET", "/api/v1/self/getSelf", rubiconCheckToken},
		{"rubiconSignoutToken", "GET", "/api/v1/authentication/signoutToken", rubiconSignoutToken},

		// OAuth Compatibility
		{"oauthToken", "POST", "/v1/profile/oauth.token", oauthToken},
		{"captcha", "GET", "/v1/profile/captcha/new", captchaGen},
		{"captcha", "GET", "/v1/profile/captcha/json", captchaJson},
		{"captchaCode", "GET", "/v1/profile/captcha/{path}", captchaCode},
		{"captchaTest", "GET", "/v1/profile/captcha/{captcha}/{code}", captchaTest},
	})

	httpsrv.IdentRegister("oauth", httpsrv.IdentRoutes{
		{"getCheckAuth", "GET", "/v1/profile/user.checkauth", getCheckAuth},
		{"oauthClient", "GET", "/v1/profile/oauth.authorize", oauthClient},
		{"oauthAuthorize", "POST", "/v1/profile/oauth.authorize", oauthAuth},
		{"oauthUser", "GET", "/v1/profile/oauth.user", oauthUser},
		{"putRegister", "POST", "/v1/profile/user.passwd", postPasswd},
	})
}

func captchaGen(w http.ResponseWriter, r *http.Request) {
	id := captcha.New()
	http.Redirect(w, r, "/profile/captcha/"+id+".png", 302)
}
func captchaJson(w http.ResponseWriter, _ *http.Request) {
	d := struct {
		Captcha string `json:"captcha"`
	}{
		captcha.New(),
	}

	writeObject(w, 201, d)
}
func captchaCode(w http.ResponseWriter, r *http.Request) {
	h := captcha.Server(240, 80)
	h.ServeHTTP(w, r)
}
func captchaTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["captcha"]
	code := vars["code"]

	if !captcha.VerifyString(id, code) {
		writeMsg(w, 401, "Invalid Code")
		return
	}

	writeMsg(w, 200, "Valid Code")
}

func getCheckAuth(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	allowAnon := false

	if h := r.Header.Get("allowAnon"); h == "true" {
		allowAnon = true
	}

	if !allowAnon && !i.IsActive() {
		writeMsg(w, http.StatusForbidden, "NO_IDENTITY")
		return
	}

	writeMsg(w, http.StatusOK, "OK")
}

func oauthAuth(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
	defer r.Body.Close()

	var req map[string]string

	var ok bool
	var err error
	var c map[string]string

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMsg(w, http.StatusInternalServerError, "ERROR: "+err.Error())
		return
	}

	client := req["client_id"]
	redirect := req["redirect_uri"]
	state := req["state"]

	if !i.IsActive() {
		writeMsg(w, http.StatusForbidden, "NOT_AUTHORIZED")
		return
	}

	ok, c, err = profile.GetHashMap("oauth-client", client)
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, "ERROR: "+err.Error())
		return
	}
	if !ok {
		writeMsg(w, http.StatusNotFound, "NOT_FOUND")
		return
	}

	if redirect != c["redirect"] {
		writeMsg(w, http.StatusBadRequest, "REDIRECT_NOMATCH")
		return
	}

	sha := sha256.Sum256([]byte(state))
	code := enc(sha[:])

	var m map[string]string
	atUser := "@" + strings.ToLower(i.GetIdentity())

	err = dbm.Transaction(func(tx *dbm.Tx) (err error) {
		if m, ok, err = model.GetHashMap(tx, atUser, client); err != nil {
			return
		}
		if ok {
			return
		}

		m = map[string]string{"ident": i.GetIdentity(), "client": client}
		err = model.PutHashMap(tx, "oauth-token", code, m)

		m = map[string]string{"code": code}
		err = model.PutHashMap(tx, atUser, "OAUTH_"+client, m)

		return
	})
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, "ERROR: "+err.Error())
		return
	}

	writeObject(w, http.StatusOK, m)
}
func oauthClient(w httpsrv.ResponseWriter, r *http.Request, _ ident.Ident) {
	client := r.URL.Query().Get("client_id")

	ok, c, err := profile.GetHashMap("oauth-client", client)
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, "ERROR: "+err.Error())
		return
	}
	if !ok {
		writeMsg(w, http.StatusNotFound, "NOT_FOUND")
		return
	}

	writeObject(w, http.StatusOK, map[string]string{"name": c["name"]})
}
func oauthToken(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var err error

	code := r.FormValue("code")
	clientId := r.FormValue("client_id")
	secret := r.FormValue("client_secret")

	var client map[string]string
	var user map[string]string
	var ok bool

	if _, client, err = profile.GetHashMap("oauth-client", clientId); err != nil {
		writeMsg(w, http.StatusInternalServerError, "ERROR: "+err.Error())
		return
	}

	if secret != client["secret"] {
		writeMsg(w, http.StatusForbidden, "WRONG_SECRET")
		return
	}

	if _, user, err = profile.GetHashMap("oauth-token", code); err != nil {
		writeMsg(w, http.StatusInternalServerError, "ERROR: "+err.Error())
		return
	}

	if clientId != user["client"] {
		writeMsg(w, http.StatusForbidden, "WRONG_CLIENT-ID")
		return
	}

	atUser := "@" + strings.ToLower(user["ident"])
	if ok, _, err = profile.GetHashMap(atUser, "OAUTH_"+clientId); err != nil {
		writeMsg(w, http.StatusInternalServerError, "ERROR: "+err.Error())
		return
	}

	if !ok {
		profile.DeleteHashMap("oauth-token", code)
		writeMsg(w, http.StatusForbidden, "TOKEN_EXPIRED")
		return
	}

	var token string
	if token, _, _, err = profile.MakeSession("default", user["ident"], profile.ProfileNone); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR:"+err.Error())
		return
	}

	o := OAuthToken{token, "bearer", 7200, code}
	log.Debug(o)

	writeObject(w, http.StatusOK, o)
}
func oauthUser(w httpsrv.ResponseWriter, _ *http.Request, i ident.Ident) {
	if !i.IsActive() {
		writeMsg(w, http.StatusForbidden, "NOT_AUTHORIZED")
		return
	}

	p, err := profile.GetUserProfile("oauth", i.GetIdentity(), profile.ProfileGlobal)
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, "ERROR: "+err.Error())
		return
	}

	o := OAuthUser{}
	o.Id = int(crc32.ChecksumIEEE([]byte(p.Ident)))
	o.Username = p.Ident
	o.Email = p.GlobalMap["mail"]
	o.Name = p.GlobalMap["displayName"]

	writeObject(w, http.StatusOK, o)
}

type OAuthUser struct {
	Id               int             `json:"id"`
	Username         string          `json:"username"`
	Email            string          `json:"email"`
	Name             string          `json:"name"`
	State            string          `json:"state"`
	AvatarUrl        string          `json:"avatar_url"`
	WebUrl           string          `json:"web_url"`
	CreatedAt        string          `json:"created_at"`
	IsAdmin          bool            `json:"is_admin"`
	Bio              string          `json:"bio"`
	Location         string          `json:"location"`
	Skype            string          `json:"skype"`
	LinkedIn         string          `json:"linkedin"`
	Twitter          string          `json:"twitter"`
	WebsiteUrl       string          `json:"website_url"`
	Organization     string          `json:"organization"`
	LastSignInAt     string          `json:"last_sign_in_at"`
	ConfirmedAt      string          `json:"confirmed_at"`
	ColorSchemeId    int             `json:"color_scheme_id"`
	ProjectsLimit    int             `json:"projects_limit"`
	CurrentSignInAt  string          `json:"current_sign_in_at"`
	Identities       []OAuthIdentity `json:"identities"`
	CanCreateGroup   bool            `json:"can_create_group"`
	CanCreateProject bool            `json:"can_create_project"`
	TwoFactorEnabled bool            `json:"two_factor_enabled"`
	External         bool            `json:"external"`
}
type OAuthIdentity struct {
	Provider string `json:"provider"`
	ExternId string `json:"extern_id"`
}
type OAuthToken struct {
	Access  string `json:"access_token"`
	Type    string `json:"token_type"`
	Expires int    `json:"expires_in"`
	Refresh string `json:"refresh_token"`
}

func putRegister(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	var err error
	var reg UserCredential
	if err = json.NewDecoder(r.Body).Decode(&reg); err != nil {
		writeMsg(w, http.StatusBadRequest, "MALFORMED_REQUEST:"+err.Error())
		return
	}
	if !captcha.VerifyString(reg.Captcha, reg.Code) {
		writeMsg(w, 401, "Invalid_Captcha")
		return
	}

	var ok bool
	if ok, err = regexp.Match(`[A-Za-z0-9\-_]+`, []byte(reg.Ident)); err != nil {
		writeMsg(w, http.StatusBadRequest, "BAD_USERNAME:"+err.Error())
		return
	}
	if !ok {
		writeMsg(w, http.StatusBadRequest, "BAD_USERNAME")
		return
	}

	ok, err = profile.CreateUser(reg.Ident, reg.Password)
	if err != nil {
		writeMsg(w, http.StatusInternalServerError, "UNKNOWN_ERR: "+err.Error())
		return
	}
	if !ok {
		writeMsg(w, http.StatusConflict, "ALREADY_REGISTERED")
		return
	}

	writeMsg(w, http.StatusCreated, "USER_REGISTERED")
}
func postPasswd(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {

	defer r.Body.Close()

	var ok bool
	var err error
	var cred UserCredential

	if err = json.NewDecoder(r.Body).Decode(&cred); err != nil {
		writeMsg(w, http.StatusBadRequest, "MALFORMED_REQUEST: "+err.Error())
		return
	}

	if cred.Ident == "" || cred.Password == "" {
		writeMsg(w, http.StatusForbidden, "AUTH_FAILED")
		return
	}

	if ok, err = profile.SetPassword(i.GetIdentity(), cred.Password); err != nil {
		writeMsg(w, http.StatusBadRequest, "ERROR: "+err.Error())
		return
	}

	if !ok {
		writeMsg(w, http.StatusForbidden, "NO_USER")
		return
	}

	writeMsg(w, http.StatusCreated, "SET_PASSWD")
}

type UserCredential struct {
	Ident    string `json:"ident"`
	Aspect   string `json:"aspect"`
	Password string `json:"password"`
	Captcha  string `json:"captcha"`
	Code     string `json:"code"`
}
type UserSession struct {
	profile.Profile
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

func postSession(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	var err error
	var cred UserCredential

	if err = json.NewDecoder(r.Body).Decode(&cred); err != nil {
		writeMsg(w, http.StatusBadRequest, "MALFORMED_REQUEST: "+err.Error())
		return
	}
	if cred.Ident == "" || cred.Password == "" {
		writeMsg(w, http.StatusForbidden, "AUTH_FAILED")
		return
	}

	var ok bool
	if ok, err = profile.CheckPassword(cred.Ident, cred.Password); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR:"+err.Error())
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
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR:"+err.Error())
		return
	}

	s := UserSession{p, key, expires.UnixNano()}
	writeObject(w, http.StatusCreated, s)
}
func deleteSession(w http.ResponseWriter, r *http.Request) {

	var ok bool
	var err error

	authorization := strings.Fields(r.Header.Get("authorization"))

	if "souris" != authorization[0] {
		w.Header().Add("www-authenticate", `souris realm="Souris Token"`)
		writeMsg(w, http.StatusUnauthorized, "Authorization Required")
		return
	}

	token := authorization[1]
	if len(authorization) > 2 {
		token = authorization[2]
	}

	if ok, err = profile.DeleteSession(token); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR: "+err.Error())
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
	Username      string
	Password      string
	GenerateToken bool
}
type RubiconUserInfo struct {
	UserId    int64  `json:"userId"`
	UserName  string `json:"userName"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}
type RubiconUserAuth struct {
	Token   string          `json:"token"`
	Expires int64           `json:"expires"`
	User    RubiconUserInfo `json:"userInfo"`
}

func rubiconAuthUser(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	var err error
	var creds RubiconCredentials
	if err = json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeMsg(w, http.StatusBadRequest, "MALFORMED_REQUEST: "+err.Error())
		return
	}

	var ok bool
	if ok, err = profile.CheckPassword(creds.Username, creds.Password); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR: "+err.Error())
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
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR: "+err.Error())
		return
	}

	ua := RubiconUserAuth{
		key,
		expires.UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		RubiconUserInfo{
			1,
			creds.Username,
			p.GlobalMap["mail"],
			p.GlobalMap["givenName"],
			p.GlobalMap["sn"],
		},
	}

	writeObject(w, http.StatusCreated, ua)
}
func rubiconCheckToken(w http.ResponseWriter, r *http.Request) {
	var ok bool
	var err error
	var p profile.Profile

	token := r.URL.Query().Get("user_token")
	if ok, p, err = profile.CheckSession("rubicon", token, profile.ProfileGlobal); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR: "+err.Error())
		return
	}
	if !ok {
		writeMsg(w, http.StatusForbidden, "AUTH_FAILED")
		return
	}

	ui := RubiconUserInfo{
		1,
		p.Ident,
		p.GlobalMap["mail"],
		p.GlobalMap["givenName"],
		p.GlobalMap["sn"],
	}

	writeObject(w, http.StatusOK, ui)
}
func rubiconSignoutToken(w http.ResponseWriter, r *http.Request) {
	var ok bool
	var err error

	token := r.URL.Query().Get("user_token")
	if ok, err = profile.DeleteSession(token); err != nil {
		writeMsg(w, http.StatusInternalServerError, "SERVER_ERROR: "+err.Error())
		return
	}
	if !ok {
		writeMsg(w, http.StatusForbidden, "NO_SESSION")
		return
	}

	writeObject(w, http.StatusNoContent, "LOGGED_OUT")
}

// Check auth

/*

func getCheckAuth(w httpsrv.ResponseWriter, r *http.Request) {
	i := ident.Ident{}


	t := AuthToken{
		i.GetIdentity(),
		i.GetAspect(),
		time.Now().UnixNano(),
		time.Now().AddDate(0,0,1).UnixNano(),
		[]byte(""),
	}
	s, _ := t.Sign(`4df1520466968debab48fc9711ac68f43e2e5a867669b8f53f5b7e6ed7df0ace`)

	w.WriteHeader(http.StatusOK)
	w.Write(s.Json())
}
func getCheckAuth2(w httpsrv.ResponseWriter, r *http.Request, i ident.Ident) {
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
	GetAspect string   `json:"aspect",bencoding:"aspect"`
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
	log.Debugf("%x %x", sk, skey_bytes)

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

func dec(s string) (dst []byte, err error) {
	src := []byte(s)
	dst = make([]byte, hex.DecodedLen(len(src)))
	_, err = hex.Decode(dst, src)
	return dst, err
}

*/

func enc(b []byte) string {
	return fmt.Sprintf("%x", b)
}
