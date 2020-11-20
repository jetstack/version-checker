package selfhosted

const (
	// Regex template to be used to check "isHost"
	hostRegTemplate = `^.*%s$`
)

func (c *Client) IsHost(host string) bool {
	return c.IsHost(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	return c.RepoImageFromPath(path)
}
