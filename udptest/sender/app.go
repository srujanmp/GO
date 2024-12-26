package main

import (

	"net"
)

func main() {
	ip := "127.0.0.1"
	port := 5000
	msg := []byte("hello world")

	addr := &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	conn, _ := net.DialUDP("udp", nil, addr)
	
	defer conn.Close()

	conn.Write(msg)
	
}
