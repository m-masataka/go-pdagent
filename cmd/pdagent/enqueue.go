package main

import (
	"net/http"
	"io"
	"strconv"
	"strings"
	"encoding/json"
	"time"
	"math/rand"
	"github.com/oklog/ulid"
	"github.com/peterbourgon/diskv"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/m-masataka/go-pdagent/pkg/pdaclient"
)

type enqueueHandler struct {
	logger  log.Logger
	disk    *diskv.Diskv
}

func (eqh *enqueueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	length, err := strconv.Atoi(r.Header.Get("Content-Length"))
	if err != nil {
		level.Error(eqh.logger).Log("msg", "Internal Server Error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var pda pdaclient.PdaEvent
	body := make([]byte, length)
	length, err = r.Body.Read(body)
	if err != nil && err != io.EOF {
		level.Error(eqh.logger).Log("msg", "Internal Server Error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.Unmarshal(body[:length], &pda); err != nil {
		level.Error(eqh.logger).Log("msg", "Internal Server Error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

        t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy).String()
	path := []string{ "in", id}
	key := strings.Join(path, "/")

	pda.Id = id

	result, err := json.Marshal(pda)
	if err != nil {
		level.Error(eqh.logger).Log("msg", "Json marshal error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := eqh.disk.Write(key, result); err != nil {
		level.Error(eqh.logger).Log("msg", "write key error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK) 
}
