package domain

const Version = "1.0.0"

type VersionInfo struct {
	Current   string
	Latest    string
	Available bool
}
