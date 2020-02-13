package main

import (
	"time"
	"context"
	"fmt"
	"strings"
	"encoding/json"
	"github.com/peterbourgon/diskv"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/m-masataka/go-pdagent/pkg/pdaclient"
)

func cleanup(ctx context.Context, d *diskv.Diskv, cleanupInterval int, cleanupTh int, logger log.Logger) {
	defer wg.Done()
	go func() {
		ticker := time.NewTicker(time.Duration(cleanupInterval) * time.Second)
	loop:

		for {
			select {
			case <-ticker.C:
				level.Info(logger).Log("msg", "cleanup mentainance started.")
				for _, prefix := range []string{"succ", "fail"} {
					for key := range d.KeysPrefix(prefix ,nil) {
						val, err := d.Read(key)
						if err != nil {
							panic(fmt.Sprintf("key %s had no value", key))
						}
						var pda pdaclient.PdaEvent
    						err = json.Unmarshal([]byte(val), &pda)
						if err != nil {
							level.Error(logger).Log("msg", "Value unmarshal error", "error", err)
							continue
						}
						if pda.Agent.QueuedAt.After(time.Now().Add(- time.Duration(cleanupTh) * time.Second)) {
							continue
						}
						d.Erase(key)
						level.Info(logger).Log("msg", "Erase Event", "Key", key, "Event", pda )
					}
				}
			case <-ctx.Done():
				level.Info(logger).Log("msg", "cleanup routine ended.")
				break loop
			}
		}
	}()
}


func backoff(ctx context.Context, d *diskv.Diskv, backoffInterval int, retryLimit int, logger log.Logger) {
	defer wg.Done()
	go func() {
		ticker := time.NewTicker(time.Duration(backoffInterval) * time.Second)
	loop:

		for {
			select {
			case <-ticker.C:
				for key := range d.KeysPrefix("fail" ,nil) {
					val, err := d.Read(key)
					if err != nil {
						panic(fmt.Sprintf("key %s had no value", key))
					}
					var pda pdaclient.PdaEvent
    					err = json.Unmarshal([]byte(val), &pda)
					if err != nil {
						level.Error(logger).Log("msg", "Value unmarshal error", "error", err)
						continue
					}
					if pda.Agent.Retry > retryLimit {
						continue
					}
					pda.Agent.Retry = pda.Agent.Retry + 1
					result, err := json.Marshal(pda)
					if err != nil {
						level.Error(logger).Log("msg", "Json marshal error", "error", err)
						continue
					}
					d.Write(strings.Replace(key, "fail", "in", 1), result)
					d.Erase(key)
					level.Info(logger).Log("msg", fmt.Sprintf("Retry ID: %s", key))
				}
			case <-ctx.Done():
				level.Info(logger).Log("msg", "backoff routine ended.")
				break loop
			}
		}
	}()
}


