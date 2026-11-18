package util

import "encoding/json"

// LoginBody returns a correctly escaped JSON login body.
func LoginBody(user, pass string) []byte {
	b, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	return b
}
