package main

import (
	"context"
	"fmt"
	"time"
        "strings"
	"encoding/json"
	"github.com/peterbourgon/diskv"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/m-masataka/go-pdagent/pkg/pdaclient"
)

func notifier(ctx context.Context, d *diskv.Diskv, interval int, logger log.Logger) {
	defer wg.Done()
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
	loop:
		for {
			select {
			case <-ticker.C:
				for key := range d.KeysPrefix("in" ,nil) {
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
					level.Info(logger).Log("key", key, "val", pda)
					resp, err := pagerduty.CreateEvent(pda.Event)
					if err != nil {
						level.Error(logger).Log("msg", "Create Event error", "error", err)
						var customRes pagerduty.EventResponse
						customRes.Status = "Notification Error"
						customRes.Message = err.Error()
						pda.EventResponse = customRes
						result, err := json.Marshal(pda)
						if err != nil {
							level.Error(logger).Log("msg", "Json marshal error", "error", err)
							continue
						}
						d.Write(strings.Replace(key, "in", "fail", 1), result)
						d.Erase(key)
						continue
					}
					pda.EventResponse = *resp
					result, err := json.Marshal(pda)
					if err != nil {
						level.Error(logger).Log("msg", "Json marshal error", "error", err)
						continue
					}
					d.Write(strings.Replace(key, "in", "succ",1 ), result)
					d.Erase(key)
				}
			case <-ctx.Done():
				level.Info(logger).Log("msg", "notifier routine ended.")
				break loop
			}
		}
	}()	
}

