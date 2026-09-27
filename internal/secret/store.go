package secret

// Store is a platform secret backend (Keychain, Windows Credential Manager, and similar).
type Store interface {
	Available() bool
	Load(account string) (string, error)
	Store(account, value string, force bool) error
	Delete(account string) error
}

// DefaultStore returns the platform secret store when supported.
func DefaultStore() Store {
	return platformStore{}
}

type platformStore struct{}

func (platformStore) Available() bool { return platformAvailable() }

func (platformStore) Load(account string) (string, error) {
	return platformLoad(account)
}

func (platformStore) Store(account, value string, force bool) error {
	return platformStoreSecret(account, value, force)
}

func (platformStore) Delete(account string) error {
	return platformDelete(account)
}
