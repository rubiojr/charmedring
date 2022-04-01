package sshproxy

import (
	"fmt"
	"io"
	"net"
	"sync"

	mlog "github.com/rubiojr/charmedring/internal/log"
)

var wg sync.WaitGroup

func Run(local, remote string) error {
	wg = sync.WaitGroup{}

	logf("listening on %s, proxying %s", local, remote)

	listener, err := net.Listen("tcp", local)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			errorf("error accepting connection %s", err)
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
		errorf("dialing remote addr", err)
		return
	}
	defer conn2.Close()

	wg.Add(2)
	go copy(conn2, conn)
	go copy(conn, conn2)
	wg.Wait()

	logf("connection to %s closed", conn.RemoteAddr())
}

func logf(msg string, args ...interface{}) {
	mlog.Infof(fmt.Sprintf("[ssh] %s", msg), args...)
}

func errorf(msg string, args ...interface{}) {
	mlog.Errorf(fmt.Sprintf("[ssh] %s", msg), args...)
}

func debugf(msg string, args ...interface{}) {
	mlog.Debugf(fmt.Sprintf("[ssh] %s", msg), args...)
}

func copy(dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
	if err != nil {
		logf("error %s", err)
	}
	wg.Done()
}
