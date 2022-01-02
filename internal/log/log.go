package log

import (
	"fmt"
	"log"
	stdlog "log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RenderError(c *gin.Context, msg string) {
	stdlog.Println(msg)
	c.String(http.StatusInternalServerError, "")
}

func RenderErrorf(c *gin.Context, msg string, args ...interface{}) {
	stdlog.Printf(msg, args...)
	c.String(http.StatusInternalServerError, "")
}

func Errorf(msg string, args ...interface{}) {
	stdlog.Printf(msg, args...)
}

func Debugf(msg string, args ...interface{}) {
	stdlog.Printf(msg, args...)
}

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print("[CRING] " + time.Now().UTC().Format("2006/01/02 - 15:04:05") + " " + string(bytes))
}

func init() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))
	log.Println("This is something being logged!")

}
