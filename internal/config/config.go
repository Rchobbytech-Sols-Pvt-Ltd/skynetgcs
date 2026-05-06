package config

const (
	SupabaseURL    = "https://nwtvgdcynvcntmrwgefl.supabase.co"
	SupabaseAPIKey = "sb_publishable_KHpfCmBYjl2oTA_PDePrCA_BND5N2RQ"
	ActivateRoute  = "/functions/v1/activate"

	GitHubOwner = "jhakrishan20"
	GitHubRepo  = "skynetgcs"

	AppName    = "SkynetGCS"
	AppVersion = "0.1.0"
)

type Component struct {
	AssetName string
	Subdir    string
	Exe       string
}

var Components = []Component{
	{AssetName: "airunit.zip", Subdir: "airunit", Exe: "airunit.exe"},
	{AssetName: "hci.zip", Subdir: "hci", Exe: "hci.exe"},
}
