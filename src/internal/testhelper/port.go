package testhelper

import (
	"log"
	"net"
)

func GetFreePort() uint16 {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}

	defer l.Close() //nolint:errcheck
	return uint16(l.Addr().(*net.TCPAddr).Port)
}
