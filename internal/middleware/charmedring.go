package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"log"

	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash"
	"github.com/gin-gonic/gin"
	mlog "github.com/rubiojr/charmedring/internal/log"
)

const partitionCount = 1069
const replicationFactor = 100
const load = 1.25

type member string

func (m member) String() string {
	return string(m)
}

type hasher struct{}

func (h hasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

func CharmedRing(urls []string) gin.HandlerFunc {
	crInfo("init")
	ring := initRing(urls)

	return func(c *gin.Context) {
		var wg sync.WaitGroup
		var count uint64

		members, err := ring.GetClosestN([]byte(c.Request.URL.Path), len(urls))
		if err != nil {
			panic(err)
		}

		rb := &binding{}
		err = c.ShouldBindBodyWith(struct{}{}, rb)
		if err != nil {
			crErrorf("failed reading request body %s", err)
			return
		}

		crDebugf("uploading %d bytes", rb.buf.Len())
		for _, u := range members {
			wg.Add(1)
			go func(m member) {
				defer wg.Done()
				if err := sendRequest(c.Request, rb.buf.Bytes(), m); err == nil {
					atomic.AddUint64(&count, 1)
				} else {
					crErrorf("sending request to %s failed: %s", m, err)
				}
			}(u.(member))
		}

		wg.Wait()
		if count < 2 {
			crErrorf("replication failed, missed %d node(s)", len(urls)-int(count))
			c.String(http.StatusServiceUnavailable, "")
			return
		}
	}
}

func sendRequest(r *http.Request, buf []byte, m member) error {
	crDebugf("sending data to %s", m.String())
	curl, err := url.Parse(m.String())
	if err != nil {
		log.Printf("error parsing charm url: %w", err)
		return err
	}
	proxyReq := r.Clone(context.Background())
	proxyReq.Body = ioutil.NopCloser(bytes.NewReader(buf))
	proxyReq.RequestURI = ""
	proxyReq.URL.Host = curl.Host
	proxyReq.URL.Scheme = curl.Scheme

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	resp.Body.Close()

	return err
}

func initRing(members []string) *consistent.Consistent {
	cfg := consistent.Config{
		PartitionCount:    partitionCount,
		ReplicationFactor: replicationFactor,
		Load:              load,
		Hasher:            hasher{},
	}

	r := consistent.New(nil, cfg)
	for _, m := range members {
		crDebugf("adding member %s to the ring", m)
		r.Add(member(m))
	}

	return r
}

func crInfo(msg string, args ...interface{}) {
	mlog.Infof(fmt.Sprintf("[cring] %s", msg), args...)
}

func crErrorf(msg string, args ...interface{}) {
	mlog.Errorf(fmt.Sprintf("[cring] %s", msg), args...)
}

func crDebugf(msg string, args ...interface{}) {
	mlog.Debugf(fmt.Sprintf("[cring] %s", msg), args...)
}
