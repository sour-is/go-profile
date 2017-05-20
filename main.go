package main

import (
	"os"
	"os/signal"
	"sour.is/x/log"
	"syscall"

	"sour.is/x/httpsrv"
	_ "sour.is/x/profile/internal/route"
	_ "sour.is/x/uuid/routes"

	"sour.is/x/profile/internal/ldap"
)

func main() {
	initConfig()

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
