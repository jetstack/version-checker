package docker

import "github.com/jetstack/version-checker/pkg/api"

type AuthResponse struct {
	Token string `json:"token"`
}

type TagResponse struct {
	Next    string   `json:"next"`
	Results []Result `json:"results"`
}

type Result struct {
	Name      string  `json:"name"`
	Timestamp string  `json:"last_updated"`
	Images    []Image `json:"images"`
}

type Image struct {
	Digest       string           `json:"digest"`
	OS           api.OS           `json:"os"`
	Architecture api.Architecture `json:"Architecture"`
}
