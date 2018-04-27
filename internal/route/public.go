package route

import "sour.is/x/toolbox/httpsrv"

func init() {
	httpsrv.AssetRegister("profile", httpsrv.AssetRoutes{
		{Name: "Assets", Path: "/", HandlerFunc: httpsrv.FsHtml5(assetFS())},
	})
}

//go:generate go-bindata-assetfs -pkg route -prefix ../../ ../../public/ ../../public/ui
