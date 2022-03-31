package sshproxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	mlog "github.com/rubiojr/charmedring/internal/log"
)

func Run(local, remote string) error {
	logf("listening on %s, proxying %s", local, remote)

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
	logf("new connection to: %s", conn.RemoteAddr())
	defer conn.Close()
	conn2, err := net.Dial("tcp", remote)
	if err != nil {
		logf("error dialing remote addr", err)
		return
	}
	defer conn2.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go copy(wg, conn2, conn)
	go copy(wg, conn, conn2)
	wg.Wait()

	logf("connection to %s closed", conn.RemoteAddr())
}

func logf(msg string, args ...interface{}) {
	mlog.Debugf(fmt.Sprintf("[ssh] %s", msg), args...)
}

func copy(wg *sync.WaitGroup, dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
	if err != nil {
		logf("error %s", err)
	}
	wg.Done()
}
