package docker

import (
	"net/http"

	"github.com/jetstack/version-checker/pkg/api"
)

type Options struct {
	Username string
	Password string
	Token    string
}

type Client struct {
	*http.Client
	Options
}

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
	Digest    string  `json:"digest"`
	Images    []Image `json:"images"`
}

type Image struct {
	Digest       string           `json:"digest"`
	OS           api.OS           `json:"os"`
	Architecture api.Architecture `json:"Architecture"`
}
