package main // import "sour.is/x/profile"

import (
	"os"
	"os/signal"
	"syscall"

	"sour.is/x/toolbox/log"

	"sour.is/x/toolbox/httpsrv"
	_ "sour.is/x/toolbox/uuid/routes"
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

		ldap.Shutdown()
	}
}
