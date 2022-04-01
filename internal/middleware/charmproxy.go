package middleware

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"

	mlog "github.com/rubiojr/charmedring/internal/log"
)

func CharmProxy(charmURL string) (gin.HandlerFunc, error) {
	var err error
	curl, err := url.Parse(charmURL)
	if err != nil {
		mlog.Errorf("[proxy] failed parsing charm URL: %s", err)
		return nil, err
	}

	director := func(req *http.Request) {
		req.URL.Host = curl.Host
		req.URL.Scheme = curl.Scheme
	}

	proxy := &httputil.ReverseProxy{Director: director}

	return func(c *gin.Context) {
		mlog.Debugf("[proxy] proxying request to %s", curl.String())
		proxy.ServeHTTP(c.Writer, c.Request)
	}, nil
}
