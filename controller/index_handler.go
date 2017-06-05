package controller

import (
	"net/http"
)

type IndexHandler struct {}

func (self *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeRsp(w, nil)
}
