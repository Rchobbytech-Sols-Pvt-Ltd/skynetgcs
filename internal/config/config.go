package config

const (
	SupabaseURL    = "https://nwtvgdcynvcntmrwgefl.supabase.co"
	SupabaseAPIKey = "sb_publishable_KHpfCmBYjl2oTA_PDePrCA_BND5N2RQ"
	ActivateRoute  = "/functions/v1/activate"

	GitHubOwner = "Rchobbytech-Sols-Pvt-Ltd"
	GitHubRepo  = "skynetgcs"

	AppName    = "SkynetGCS"
	AppVersion = "0.1.5"
)

// Component declares one piece of the GCS bundle that lives as a separate
// release asset and gets extracted into its own subdir next to the launcher.
//
// AssetPrefix is matched against release-asset filenames as a
// case-insensitive prefix; the matching asset must also end in ".zip".
// This lets release artifacts include version suffixes like
// "AirUnit v1.2.2-alpha.zip" or "AirUnit v1.3.0.zip" without changing
// the launcher.
type Component struct {
	AssetPrefix string
	Subdir      string
	Exe         string
}

var Components = []Component{
	// {AssetPrefix: "AirUnit", Subdir: "airunit", Exe: "airunit.exe"},
	// {AssetPrefix: "HCI", Subdir: "hci", Exe: "hci.exe"},
	{AssetPrefix: "AirUnit", Subdir: "airunit", Exe: "airunit/airunit.exe"},
	{AssetPrefix: "Human_Computer_Interface", Subdir: "Human_Computer_Interface", Exe: "appHuman_Computer_interface.exe"},
}
