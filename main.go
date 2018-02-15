package main // import "sour.is/x/profile"

import (
	"os"
	"os/signal"
	"syscall"

	"sour.is/go/log"

	"sour.is/go/httpsrv"
	_ "sour.is/go/uuid/routes"
	_ "sour.is/x/profile/internal/route"

	"sour.is/x/profile/internal/ldap"
)

func main() {
	if args["serve"] == true {
		go httpsrv.Run()
		go ldap.Run()

		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		close(ch)

		log.Notice("Shutting Down Server")

		httpsrv.Shutdown()
		ldap.Shutdown()
	}
}
