package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"github.com/go-kit/kit/log"
	"github.com/peterbourgon/diskv"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"

	"github.com/m-masataka/go-pdagent/pkg/pdaclient"
)

type searchHandler struct {
	logger  log.Logger
	disk    *diskv.Diskv
}

func (sh *searchHandler) keyToPda(key string) (pdaclient.PdaEvent, error){
	var pda pdaclient.PdaEvent
	val, err := sh.disk.Read(key)
	if err != nil {
		return pda, err
	}
    	err = json.Unmarshal([]byte(val), &pda)
	if err != nil {
		return pda, err
	}
	return pda, nil
}

func (sh *searchHandler) getValByPrefix(prefix string) ([]pdaclient.PdaEvent, error){
	var pdas []pdaclient.PdaEvent
	for key := range sh.disk.KeysPrefix(prefix, nil) {
		pda, err := sh.keyToPda(key)
		if err != nil {
			return pdas, err
		}
		pdas = append(pdas, pda)
	}
	return pdas, nil
}

func (sh *searchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	level.Debug(sh.logger).Log("msg", fmt.Sprintf("Search API prefix : %s", vars["prefix"]))
	prefix := vars["prefix"]
	resPda, err := sh.getValByPrefix(prefix)
	if err != nil {
		level.Error(sh.logger).Log("msg", "Value unmarshal error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := json.Marshal(resPda)
	if err != nil {
		level.Error(sh.logger).Log("msg", "Value unmarshal error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}
