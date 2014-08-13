package pickle

import (
	"encoding/binary"
	"github.com/mattrobenolt/mineshaft/metric"
	"github.com/mattrobenolt/mineshaft/store"
	"log"
	"net"
)

type Pickle struct {
	*store.Store
}

func (s *Pickle) recv(c net.Conn) {
	var point = metric.New()
	defer c.Close()
	defer metric.Release(point)

	// read the length header
	var size int32
	binary.Read(c, binary.BigEndian, &size)

	decoder := NewDecoder(c)
	data, err := decoder.Decode()
	if err != nil {
		log.Println("Error decoding pickle data", err)
		return
	}

	for _, item := range data.([]interface{}) {
		metric := item.([]interface{})
		values := metric[1].([]interface{})

		point.Path = metric[0].(string)
		point.Timestamp = uint32(values[0].(int64))

		// based on if the value was an integer or a float, the
		// value may be either an int64 or a float64
		switch v := values[1].(type) {
		case int64:
			point.Value = float64(v)
		case float64:
			point.Value = v
		}

		s.Store.Set(point)
	}
}

func ListenAndServe(addr string, s *store.Store) error {
	log.Println("Starting pickle on", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	c := &Pickle{s}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go c.recv(conn)
	}

	return nil
}
