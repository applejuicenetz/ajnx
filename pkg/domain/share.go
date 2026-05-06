package domain

// Share repräsentiert eine einzelne freigegebene Datei im appleJuice Netzwerk.
type Share struct {
	ID            string
	Hash          string
	Size          int64
	Filename      string
	ShortFilename string
	Priority      int
}

// ShareDir repräsentiert ein freigegebenes Verzeichnis im Core.
type ShareDir struct {
	Path           string
	WithSubfolders bool
}
