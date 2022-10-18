package main

import (
	"errors"
	"net"
)

func dialAndNegotiateAuto(addr string, baseDial func() (net.Conn, error)) (net.Conn, error) {
	return nil, errors.New("auto detect ntlm creds not supported on linux")
}
