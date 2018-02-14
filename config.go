package main

import (
	"bytes"
	"fmt"
	"github.com/docopt/docopt.go"
	"github.com/spf13/viper"
	"sour.is/x/dbm"
	"sour.is/x/httpsrv"
	"sour.is/x/log"
	_ "sour.is/x/profile/internal/ident"
	"sour.is/x/profile/internal/ldap"
)

var (
	APP_VERSION string
	APP_BUILD   string
)

var APP_NAME string = "Souris Profile API"
var APP_USAGE string = `Souris Profile API

Usage:
  profile version
  profile [ -v | -vv ] serve

Options:
  -v                                             Log info to console.
  -vv                                            Log debug to console.
  -l <ListenAddress>, --listen=<ListenAddress>   Address to listen on.
  -c <ConfigDir>, --config=<ConfigDir>           Set Config Directory.

Config:
  The config file is read from the following locations:
    - <ConfigDir>
    - /etc/opt/sour.is/profile/
    - Working Directory
`
var defaultConfig string = `
database   = "local"

[db.local]
type      = "mysql"
connect   = "profile:profile@tcp(127.0.0.1:3306)/profile"

[http]
listen   = ":8060"
identity = "souris"

[ldap]
listen = ":3389"
baseDN = "dc=sour,dc=is"
domain = "sour.is"
`

var args map[string]interface{}

func initConfig() {
	var err error

	if args, err = docopt.Parse(APP_USAGE, nil, true, APP_NAME, false); err != nil {
		log.Fatal(err)
	}

	if args["-v"].(int) == 1 {
		log.SetVerbose(log.Vinfo)
	}
	if args["-v"].(int) == 2 {
		log.SetVerbose(log.Vdebug)
		log.Notice("Debug Logging.")
	}

	viper.SetConfigName("config")
	if args["--config"] != nil {
		viper.AddConfigPath(args["--config"].(string))
	}

	viper.AddConfigPath("/etc/opt/sour.is/profile/")
	viper.AddConfigPath(".")

	viper.SetConfigType("toml")
	viper.ReadConfig(bytes.NewBuffer([]byte(defaultConfig)))

	err = viper.MergeInConfig()
	if err != nil { // Handle errors reading the config file
		log.Fatalf("Fatal error config file: %s \n", err)
	}

	viper.Set("app.name", APP_NAME)

	viper.SetDefault("app.version", "VERSION")
	if APP_VERSION != "" {
		viper.Set("app.version", APP_VERSION)
	}

	viper.SetDefault("app.build", "SNAPSHOT")
	if APP_BUILD != "" {
		viper.Set("app.build", APP_BUILD)
	}

	if args["serve"] == true {

		if args["--listen"] != nil {
			viper.Set("listen", args["--listen"].(string))
		}

		log.Noticef("Startup: %s (%s %s)",
			viper.GetString("app.name"),
			viper.GetString("app.version"),
			viper.GetString("app.build"))

		log.Notice("Read config from: ", viper.ConfigFileUsed())

		dbm.Config()
/*
		if err = dbm.Migrate(dbm.Asset{File: Asset, Dir: AssetDir}); err != nil {
			panic(err)
		}
*/
		if viper.IsSet("http") {
			httpsrv.Config()
		}
		if viper.IsSet("ldap") {
			ldap.Config()
		}

	}

	if args["version"] == true {
		fmt.Printf("Version: %s (%s %s)\n",
			viper.GetString("app.name"),
			viper.GetString("app.version"),
			viper.GetString("app.build"))
	}

}
