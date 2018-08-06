package ident

import "net/http"
import (
	"strings"

	"sour.is/x/toolbox/ident"
	"sour.is/x/toolbox/log"
	"sour.is/x/profile/internal/profile"
)

func init() {
	ident.Register("souris", NewUser)
}

type User struct {
	ident  string
	aspect string
	name   string
	roles  map[string]bool
	groups map[string]bool
	active bool
}

var anon User = User{
	"anon",
	"default",
	"Guest User",
	make(map[string]bool),
	make(map[string]bool),
	false,
}

func NewUser(r *http.Request) ident.Ident {

	authorization := strings.Fields(r.Header.Get("authorization"))

	var token, authType, aspect string

	switch len(authorization) {
	case 3:
		authType = authorization[0]
		aspect = authorization[1]
		token = authorization[2]

		break
	case 2:
		authType = authorization[0]
		aspect = "default"
		token = authorization[1]

		break
	default:
		return anon
	}

	switch authType {
	case "Bearer":
	case "souris":
		break
	default:
		return anon
	}

	roles := make(map[string]bool)
	groups := make(map[string]bool)

	log.Debugf("auth Token: [%s] Aspect: [%s]", token, aspect)

	var ok bool
	var p profile.Profile
	var err error

	if ok, p, err = profile.CheckSession(aspect, token, profile.ProfileNone); err != nil || !ok {
		log.Debug("Invalid Session")
		return anon
	}

	name := p.GlobalMap["display_name"]
	if name == "" {
		name = p.Ident
	}

	for _, n := range p.Roles {
		roles[n] = true
	}
	for _, n := range p.Groups {
		groups[n] = true
	}

	log.Debugf("%+v", p)

	return User{
		p.Ident,
		p.Aspect,
		name,
		roles,
		groups,
		true,
	}
}

func (m User) GetIdentity() string {
	return m.ident
}

func (m User) GetAspect() string {
	return m.aspect
}

func (m User) HasRole(r ...string) (ok bool) {
	for _, n := range r {
		if _, ok = m.roles[n]; ok {
			break
		}
	}
	return
}

func (m User) HasGroup(g ...string) (ok bool) {
	for _, n := range g {
		if _, ok = m.roles[n]; ok {
			break
		}
	}
	return
}

func (m User) IsActive() bool {
	return m.active
}

func (m User) GetDisplay() string {
	return m.name
}
