package domain

// Search stellt eine aktive Suche im Netzwerk dar.
type Search struct {
	ID          string         `json:"id"`
	Query       string         `json:"query"`
	FoundFiles  int            `json:"foundFiles"`
	SumSearches int            `json:"sumSearches"`
	Running     bool           `json:"running"`
	Results     []SearchResult `json:"results,omitempty"`
}

// SearchResult stellt eine gefundene Datei dar.
type SearchResult struct {
	ID       string             `json:"id"`
	SearchID string             `json:"searchId"`
	Hash     string             `json:"hash"`
	Size     int64              `json:"size"`
	Names    []SearchResultName `json:"names"`
}

// SearchResultName enthält einen Dateinamen und wie oft er gefunden wurde.
type SearchResultName struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
