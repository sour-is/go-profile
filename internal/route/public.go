package route

import "sour.is/go/httpsrv"

func init() {
	httpsrv.AssetRegister("profile", httpsrv.AssetRoutes{
		{Name: "Assets", Path: "/", HandlerFunc: httpsrv.FsHtml5(assetFS())},
	})
}
