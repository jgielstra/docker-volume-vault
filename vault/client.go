package vault

import (
	"crypto/tls"
	"net/http"

	"github.com/hashicorp/vault/api"
)

var DefaultConfig *api.Config

// NewConfig creates a new config
func NewConfig(address string, insecure bool) *api.Config {
	return &api.Config{
		Address: address,
		HttpClient: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecure,
				},
			},
		},
	}
}

func Client(token string) (*api.Client, error) {
	client, err := api.NewClient(DefaultConfig)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)
	return client, nil
}
