/*
Copyright © 2021 Sergio Rubio <sergio@rubio.im>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rubiojr/charmedring/internal/middleware"
	"github.com/rubiojr/charmedring/internal/sshproxy"
	"github.com/spf13/cobra"
)

var hosts []string
var addr string
var sshBackendPort int
var sshProxyAddr string

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the proxy server",
	RunE: func(cmd *cobra.Command, args []string) error {
		gin.SetMode(gin.ReleaseMode)
		router := gin.Default()
		if len(hosts) == 0 {
			fmt.Println("No hosts specified.")
			os.Exit(1)
		}

		sshURL, err := url.Parse(hosts[0])
		if err != nil {
			fmt.Println("Error parsing host:", err)
			return err
		}

		cring := middleware.CharmedRing(hosts)
		authorized := router.Group("/")
		authorized.GET("/*path", middleware.CharmProxy(hosts[0]))
		authorized.POST("/v1/fs/*path", cring)
		authorized.DELETE("/v1/fs/*path", cring)
		router.NoRoute(middleware.CharmProxy(hosts[0]))

		log.Printf("SSH Proxy listening on %s", sshProxyAddr)
		sshBackend := fmt.Sprintf("%s:%d", sshURL.Hostname(), sshBackendPort)
		log.Printf("SSH backend on %s", sshBackend)
		go sshproxy.Run(sshProxyAddr, sshBackend)
		log.Printf("listening on %v\n", addr)
		return router.Run(addr)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringArrayVar(&hosts, "host", []string{}, "Hosts to proxy to")
	serveCmd.Flags().StringVar(&addr, "address", ":35354", "Listen address")
	serveCmd.Flags().IntVar(&sshBackendPort, "ssh-backend-port", 35353, "SSH backend port")
	serveCmd.Flags().StringVar(&sshProxyAddr, "ssh-proxy-address", ":35353", "SSH proxy address")
}
