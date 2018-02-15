package profile

import (
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"

	"github.com/spf13/viper"
	"golang.org/x/crypto/pbkdf2"
)

type RegAuthCreds struct {
	Username string
	Password string
}

func PassSiska(user, pass, secret string) (ok bool, check string) {
	gsh_salt1 := viper.GetString("auth.siska.salt1")
	gsh_salt2 := viper.GetString("auth.siska.salt2")
	gsh_salt3 := viper.GetString("auth.siska.salt3")
	gsh_salt4 := viper.GetString("auth.siska.salt4")

	p := sha512.Sum512([]byte(gsh_salt4 + pass + user + gsh_salt1 + gsh_salt3 + gsh_salt2))
	s := hex.EncodeToString(p[:])
	s = base64.StdEncoding.EncodeToString([]byte(s))

	k := pbkdf2.Key([]byte(s), []byte(gsh_salt3), 2048, 256, sha512.New)
	h := hex.EncodeToString(k)

	check = base64.StdEncoding.EncodeToString([]byte(h))

	if secret != "" {
		if subtle.ConstantTimeCompare([]byte(secret), []byte(check)) == 1 {
			ok = true
		}
	}

	return
}
