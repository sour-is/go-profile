package route

import "sour.is/x/httpsrv"

func init() {
	httpsrv.AssetRegister("profile", httpsrv.AssetRoutes{
		{ "Assets", "/", httpsrv.FsHtml5( assetFS() ) },
	})
}