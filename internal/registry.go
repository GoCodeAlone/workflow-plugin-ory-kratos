package internal

import (
	"sync"

	kratos "github.com/ory/kratos-client-go"
)

type OryKratosClient struct {
	Admin     *kratos.APIClient
	Public    *kratos.APIClient
	AdminURL  string
	PublicURL string
}

var (
	clientMu       sync.RWMutex
	clientRegistry = map[string]*OryKratosClient{}
)

func RegisterClient(name string, client *OryKratosClient) {
	clientMu.Lock()
	defer clientMu.Unlock()
	clientRegistry[name] = client
}

func GetClient(name string) (*OryKratosClient, bool) {
	clientMu.RLock()
	defer clientMu.RUnlock()
	client, ok := clientRegistry[name]
	return client, ok
}

func UnregisterClient(name string) {
	clientMu.Lock()
	defer clientMu.Unlock()
	delete(clientRegistry, name)
}
