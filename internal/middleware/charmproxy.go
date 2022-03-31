package middleware

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"

	mlog "github.com/rubiojr/charmedring/internal/log"
)

func CharmProxy(charmURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		curl, err := url.Parse(charmURL)
		if err != nil {
			mlog.RenderError(c, fmt.Sprintf("error parsing charm url: %s", err))
			return
		}

		mlog.Debugf("proxying request: %s", curl.String())
		director := func(req *http.Request) {
			req.URL.Host = curl.Host
			req.URL.Scheme = curl.Scheme
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
