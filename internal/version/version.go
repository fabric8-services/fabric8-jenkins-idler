package version

var (
	version = "unset"
)

// GetVersion gets you the current version of Jenkins Idler
func GetVersion() string {
	return version
}
