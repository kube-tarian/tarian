package version

var versionStr string

func SetVersion(ver string) {
	versionStr = ver
}

func GetVersion() string {
	return versionStr
}
