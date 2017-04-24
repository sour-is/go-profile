package main

import (
	"os"
	"os/signal"
	"syscall"
	"sour.is/x/log"

	"sour.is/x/httpsrv"
	_ "sour.is/x/uuid/routes"
	_ "sour.is/x/profile/internal/route"

	"sour.is/x/profile/internal/ldap"
)

func main() {
	initConfig()

	if args["serve"] == true{
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
