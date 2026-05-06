package domain

const Version = "1.0.1-beta"

type VersionInfo struct {
	Current   string
	Latest    string
	Available bool
}
