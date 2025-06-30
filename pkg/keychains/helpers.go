package keychains

import (
	"slices"

	v1 "k8s.io/api/core/v1"
)

// The `pullSecrets` function takes a slice of `LocalObjectReference` objects,
// extracts their names, sorts them alphabetically, removes duplicates and returns a slice of strings.
func pullSecrets(secrets []v1.LocalObjectReference) (pullSecrets []string) {
	for _, sec := range secrets {
		pullSecrets = append(pullSecrets, sec.Name)
	}
	// Sort the list of Secrets
	slices.Sort(pullSecrets)
	// Remove duplicates
	uniquePullSecrets := make([]string, 0, len(pullSecrets))
	seen := make(map[string]struct{})
	for _, secret := range pullSecrets {
		if _, ok := seen[secret]; !ok {
			seen[secret] = struct{}{}
			uniquePullSecrets = append(uniquePullSecrets, secret)
		}
	}
	return uniquePullSecrets
}

// The function `saName` returns the input string if it is not empty, otherwise it
// returns "default".
func saName(saName string) string {
	if saName == "" {
		return "default"
	}
	return saName
}
