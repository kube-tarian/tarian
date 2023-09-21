package version

var versionStr string

// SetVersion sets the version string
func SetVersion(ver string) {
	versionStr = ver
}

// GetVersion returns the version string
func GetVersion() string {
	return versionStr
}
