package main

import (
	"bytes"
	"fmt"

	"github.com/docopt/docopt.go"
	"github.com/spf13/viper"

	"sour.is/x/toolbox/dbm"
	"sour.is/x/toolbox/httpsrv"
	"sour.is/x/toolbox/log"

	_ "github.com/go-sql-driver/mysql"
	_ "sour.is/x/profile/internal/ident"

	"sour.is/x/profile/internal/ldap"
)

var (
	// AppVersion Application Version Number
	AppVersion string

	// AppBuild Application Build Number
	AppBuild string
)

// AppName name of the application
var AppName = "Souris Profile API"

// AppUsage Application Usage
var AppUsage = `Souris Profile API

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
var defaultConfig = `
database   = "local"

[db.local]
type      = "mysql"
connect   = "profile:profile@tcp(127.0.0.1:3306)/profile"
migrate   = "true"

[http]
listen   = ":8060"
identity = "souris"

[ldap]
listen = ":3389"
baseDN = "dc=sour,dc=is"
domain = "sour.is"
`

var args map[string]interface{}

func init() {
	var err error

	if args, err = docopt.Parse(AppUsage, nil, true, AppName, false); err != nil {
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

	viper.Set("app.name", AppName)

	viper.SetDefault("app.version", "VERSION")
	if AppVersion != "" {
		viper.Set("app.version", AppVersion)
	}

	viper.SetDefault("app.build", "SNAPSHOT")
	if AppBuild != "" {
		viper.Set("app.build", AppBuild)
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

		migrate := false
		if viper.IsSet("database") {
			pfx := "db." + viper.GetString("database") + ".migrate"
			migrate = viper.GetBool(pfx)
		}

		if migrate {
			if err = dbm.Migrate(dbm.Asset{File: Asset, Dir: AssetDir}); err != nil {
				log.Fatal(err.Error())
			}
		}

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
