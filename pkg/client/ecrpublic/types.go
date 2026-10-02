package ecrpublic

type AuthResponse struct {
	Token string `json:"token"`
}

type TagResponse struct {
	Next string   `json:"next"`
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}
