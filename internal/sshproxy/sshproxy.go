package sshproxy

import (
	"io"
	"log"
	"net"
)

func Run(local, remote string) error {
	logf("listening: %v\nproxying: %v\n\n", local, remote)

	listener, err := net.Listen("tcp", local)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("error accepting connection", err)
			continue
		}
		go handleConn(conn, remote)
	}
}

func handleConn(conn net.Conn, remote string) {
	logf("new connection: %s", conn.RemoteAddr())
	defer conn.Close()
	conn2, err := net.Dial("tcp", remote)
	if err != nil {
		logf("error dialing remote addr", err)
		return
	}
	defer conn2.Close()
	closer := make(chan struct{}, 2)
	go copy(closer, conn2, conn)
	go copy(closer, conn, conn2)
	<-closer
	logf("connection complete", conn.RemoteAddr())
}

func logf(format string, args ...interface{}) {
	log.Printf("[SSH_PROXY] "+format, args...)
}

func copy(closer chan struct{}, dst io.Writer, src io.Reader) {
	_, _ = io.Copy(dst, src)
	closer <- struct{}{}
}
