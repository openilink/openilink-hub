package store

type ConfigStore interface {
	GetConfig(key string) (string, error)
	SetConfig(key, value string) error
	DeleteConfig(key string) error
	ListConfigByPrefix(prefix string) (map[string]string, error)
}
