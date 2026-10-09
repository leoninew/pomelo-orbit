package envfile

import file "github.com/leoninew/pomelo-orbit/internal/common/envfile"

type Store struct {
	file.Store
}

func NewStore(path string) Store {
	return Store{Store: file.NewStore(path)}
}
