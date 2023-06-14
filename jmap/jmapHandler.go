package jmap

import "net/http"

type JMAPHandler struct {
}

func NewJMAPHandler() JMAPHandler {
	return JMAPHandler{}
}

func (jh JMAPHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("jmap handler"))
}
