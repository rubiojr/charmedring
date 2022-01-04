package middleware

import (
	"bytes"
	"net/http"
)

type binding struct {
	buf *bytes.Buffer
}

func (t *binding) BindBody(b []byte, i interface{}) error {
	t.buf = bytes.NewBuffer(b)
	return nil
}

func (t *binding) Bind(req *http.Request, i interface{}) error {
	return nil
}

func (t *binding) Name() string {
	return "binding"
}
