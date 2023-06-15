package jmaphandler

import "net/http"

type JMAPServerHandler struct {
}

func NewHandler() JMAPServerHandler {
	return JMAPServerHandler{}
}

func (jh JMAPServerHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("jmap handler"))

	//need to setup routing
}
