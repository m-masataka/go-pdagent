package main

import (
	"fmt"
	"net/http"
	"strings"
	"github.com/go-kit/kit/log"
	"github.com/peterbourgon/diskv"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
)

type retryHandler struct {
	logger  log.Logger
	disk    *diskv.Diskv
}

func (rh *retryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	level.Debug(rh.logger).Log("msg", fmt.Sprintf("Retry : %s", vars["id"]))
	id := vars["id"]
	path := []string{ "fail", id}
	key := strings.Join(path, "/")
	val, err := rh.disk.Read(key)
	if err != nil {
		level.Error(rh.logger).Log("msg", "Get key error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	rh.disk.Write(strings.Replace(key, "fail", "in", 1), val)
	rh.disk.Erase(key)
	w.WriteHeader(http.StatusOK) 
}
