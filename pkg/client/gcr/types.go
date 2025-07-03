package gcr

type Response struct {
	Manifest map[string]ManifestItem `json:"manifest"`
}

type ManifestItem struct {
	Tag         []string `json:"tag"`
	TimeCreated string   `json:"timeCreatedMs"`
}
