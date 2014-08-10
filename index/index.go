package index

import (
	"net/url"
)

type Store struct {
	driver Driver
}

func (s *Store) Update(path string) error {
	return s.driver.Update(path)
}

func (s *Store) Ping() error {
	return s.driver.Ping()
}

type Driver interface {
	Init(*url.URL) error
	Update(string) error
	Ping() error
	Close()
}

func Register(key string, d Driver) {
	registry[key] = d
}

func GetDriver(url *url.URL) Driver {
	d, ok := registry[url.Scheme]
	if !ok {
		panic("index: driver not found")
	}
	err := d.Init(url)
	if err != nil {
		panic(err)
	}
	return d
}

func NewFromConnection(url *url.URL) *Store {
	d := GetDriver(url)
	return &Store{d}
}

var registry = make(map[string]Driver)
