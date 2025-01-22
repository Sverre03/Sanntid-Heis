package localip

import (
	"net"
	"strings"
)

var localIP string

// LocalIP returns the local IP address of the machine.
// It establishes a TCP connection to a remote address to determine the local IP.
func LocalIP() (string, error) {
	if localIP == "" {
		conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: []byte{8, 8, 8, 8}, Port: 53})
		if err != nil {
			return "", err
		}
		defer conn.Close()
		localIP = strings.Split(conn.LocalAddr().String(), ":")[0]
	}
	return localIP, nil
}
