package middleware

import (
	"bytes"
	"context"
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
	ring := initRing(urls)

	return func(c *gin.Context) {
		var wg sync.WaitGroup
		var count uint64

		members, err := ring.GetClosestN([]byte(c.Request.URL.Path), len(urls))
		if err != nil {
			panic(err)
		}

		buf, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			panic(err)
		}

		for _, u := range members {
			wg.Add(1)
			go func(m member) {
				defer wg.Done()
				if err := sendRequest(c.Request, buf, m); err == nil {
					atomic.AddUint64(&count, 1)
				}
			}(u.(member))
		}

		wg.Wait()
		if count < 2 {
			mlog.Errorf("replication failed, missed %d node(s)", len(urls)-int(count))
			c.String(http.StatusServiceUnavailable, "")
		}
	}
}

func sendRequest(r *http.Request, buf []byte, m member) error {
	mlog.Debugf("sending data to %s", m.String())
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
	_, err = client.Do(proxyReq)
	if err != nil {
		mlog.Debugf("error: sending request to %s failed: %s", m, err)
	}
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
		mlog.Debugf("adding member %s to the ring", m)
		r.Add(member(m))
	}

	return r
}
