package main

import (
	"fmt"
	"net"
)

func main() {
	ip := "127.0.0.1"
	port := 5000


	addr := &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	conn,_  := net.ListenUDP("udp", addr)
	
	defer conn.Close()

	buffer := make([]byte, 1024)

	for {
		n, _, _ := conn.ReadFromUDP(buffer)
		fmt.Printf("Received message: %s\n", string(buffer[:n]))
	}
}
