package hbs

import (
	"fmt"
	"github.com/dinp/server/g"
	"log"
	"net"
	"net/rpc"
)

func Start() {
	addr := fmt.Sprintf("%s:%d", g.Config().Rpc.Addr, g.Config().Rpc.Port)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatalf("net.ResolveTCPAddr fail: %s", err)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatalf("listen %s fail: %s", addr, err)
	}

	rpc.Register(new(NodeState))

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("listener.Accept occur error: %s", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}
