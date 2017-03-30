package ldap

import (
	"strings"
	"github.com/nmcclain/ldap"
	"net"
	"sour.is/x/profile/internal/profile"
	"sour.is/x/log"
	"os"
	"time"
	"github.com/spf13/viper"
)

var listen string
var domain string
var baseDN string

var accessLog = log.New(os.Stdout, "", log.Ldate | log.Ltime | log.LUTC)

type ldapHandler struct {}

func Run() {
	s := ldap.NewServer()

	handler := ldapHandler{}
	s.BindFunc("", handler)
	s.SearchFunc("", handler)

	s.EnforceLDAP = true
	log.Noticef("Starting LDAP on: %s", listen)

	if err := s.ListenAndServe(listen); err != nil {
		log.Criticalf("LDAP Server Failed: %s", err.Error())
		return
	}
}

func Config() {
	listen = viper.GetString("ldap.listen")
	domain = viper.GetString("ldap.domain")
	baseDN = viper.GetString("ldap.baseDN")
}

func (h ldapHandler) Bind(user, password string, conn net.Conn) (ldap.LDAPResultCode, error) {
	//log.Debugf("%s %s", user, password)

	user, _ = getUsername(user)
	if user != "" && password != "" {
		var ok bool
		var err error

		if ok, err = profile.CheckPassword(user, password); err != nil {
			log.Error(err)
			return ldap.LDAPResultOperationsError, err
		}
		if !ok {
			log.Debugf("AUTH FAIL: %s", user)
			return ldap.LDAPResultInvalidCredentials, nil
		}

		//log.Debug("AUTH SUCCESS")
		return ldap.LDAPResultSuccess, nil
	}

	return ldap.LDAPResultInvalidCredentials, nil
}

func getUsername(dn string) (user, aspect string) {
	user = "anon"
	aspect = "default"

	if strings.Contains(dn, "@") && strings.HasSuffix(dn, domain){
		dn = strings.TrimSuffix(dn, domain)

		at := strings.SplitN(dn, "@", 2)
		user = at[0]

		dot := strings.SplitN(at[1], ".", 2)
		if aspect != "" {
			aspect = dot[0]
		}

		return user, aspect
	}

	if strings.HasSuffix(strings.ToLower(dn), "," + baseDN) {
		c := strings.Split(dn, ",")

		for _, v := range c {
			eq := strings.SplitN(v, "=", 2)

			if strings.ToLower(eq[0]) == "cn" {
				user = eq[1]
			}

			if strings.ToLower(eq[0]) == "ou" {
				aspect = eq[1]
			}

		}

		return
	}

	return
}

func (h ldapHandler) Search(boundDN string, searchReq ldap.SearchRequest, conn net.Conn) (ldap.ServerSearchResult, error) {
	start := time.Now()

	var user string
	var aspect string

	user, aspect = getUsername(boundDN)

	p, err := profile.GetUserProfile(aspect, user, profile.ProfileGlobal)
	if err != nil {
		return ldap.ServerSearchResult{}, err
	}

	active := "inactive"
	admin := "user"
	for _, r := range p.Roles {
		if r == "active" {
			active = "active"
		}

		if r == "admin" {
			admin = "admin"
		}
	}

	entries := []*ldap.Entry{
		{"cn=" + user + "," + searchReq.BaseDN,
			[]*ldap.EntryAttribute{
				{"cn",            []string{user}},
				{"dispayName",    []string{p.GlobalMap["display_name"]}},
				{"givenName",     []string{p.GlobalMap["first_name"]}},
				{"sn",            []string{p.GlobalMap["last_name"]}},
				{"mail",          []string{p.GlobalMap["email"]}},
				{"accountStatus", []string{active}},
				{"uid",           []string{user}},
				{"memberOf",      []string{admin}},
				{"aspect",        []string{aspect}},
				{"objectClass",   []string{"person"}},
			}},
	}


	accessLog.Printf(
		"%s\t%s/%s\t% 12s\t%d\t%s\t%s %s",
		"ldapSearch",
		aspect,
		user,
		time.Since(start),
		ldap.LDAPResultSuccess,
		searchReq.BaseDN,
		searchReq.Filter,
		searchReq.Attributes,
	)

	return ldap.ServerSearchResult{
		Entries: entries,
		Referrals: []string{},
		Controls: []ldap.Control{},
		ResultCode: ldap.LDAPResultSuccess}, nil
}