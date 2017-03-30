package ident

import "net/http"
import (
	"sour.is/x/ident"
	"strings"
	"sour.is/x/profile/internal/profile"
	"sour.is/x/log"
)

func init () {
	ident.Register("souris", NewUser)
}

type User struct{
	ident string
	aspect string
	name string
	roles map[string]bool
	active bool
}

func NewUser(r *http.Request) ident.Ident {

	token := r.Header.Get("authorization")
	aspect := r.URL.Query().Get("aspect")
	if aspect == "" {
		aspect = "default"
	}

	roles := make(map[string]bool)

	if !strings.HasPrefix(token, "souris ") {
		return User{
			"anon",
			"default",
			"Guest User",
			roles,
			false,
		}
	}

	token = strings.TrimPrefix(token, "souris ")
	log.Debugf("Auth Token: [%s]", token)

	var ok bool
	var p profile.Profile
	var err error

	if ok, p, err = profile.CheckSession(aspect, token, profile.ProfileNone); err != nil || !ok {
		log.Debug("Invalid Session")

		return User{
			"anon",
			"default",
			"Guest User",
			roles,
			false,
		}
	}

	name := p.GlobalMap["display_name"]
	if name == "" {
		name = p.Ident
	}

	for _, n := range p.Roles {
		roles[n] = true
	}

	log.Debugf("%+v",p)

	return User{
		p.Ident,
		p.Aspect,
		name,
		roles,
		true,
	}
}

func (m User)Identity() string {
	return m.ident
}

func (m User)Aspect() string {
	return m.aspect
}

func (m User)HasRole(r string) bool {
	_, ok := m.roles[r]
	return ok
}

func (m User)HasGroup(g string) bool {
	return m.active
}

func (m User)LoggedIn() bool {
	return m.active
}

func (m User)DisplayName() string {
	return m.name
}