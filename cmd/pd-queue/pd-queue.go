package main

import (
	"os"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"context"
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/BurntSushi/toml"
	"github.com/m-masataka/go-pdagent/pkg/pdaclient"
)

type config struct {
	Main mainConfig `toml:"Main"`
}

type mainConfig struct {
	Socket                      string `toml:"socket"`
}

var (
	conf config
	socket string
)

func loadInit(configFile string) {
	if _, err := toml.DecodeFile(configFile, &conf); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	socket = conf.Main.Socket
}
	
	
func main() {
	var (
		app = kingpin.New("manager", "Management command")
		configFile = app.Flag("config", "Path for config file.").Default("config/pdagent.conf").String()
		statusCommand = app.Command("status", "print out status of event queue.")
		statusCommandSKey= statusCommand.Flag("service-key", "print status of events in given Service API Key").Short('k').String()
		detailCommand = app.Command("detail", "print out id list of event queue.")
		detailCommandSKey= detailCommand.Flag("service-key", "print detail of events in given Service API Key").Short('k').String()
		detailCommandId= detailCommand.Flag("id", "print detail of events in given Id").String()

		retryCommand = app.Command("retry", "Set up 'dead' pagerduty events for retry.")
		retryCommandAll = retryCommand.Flag("all-keys", "retry events in all Service API Keys (not to be used with -k)").Default("con").Short('a').String()
		retryCommandSKey = retryCommand.Flag("service-key", "retry events in given Service API Key (not to be used with all)").Short('k').String()
		retryCommandId = retryCommand.Flag("id", "retry events in given Id (not to be used with all").String()

		//retryAllCommand = retryCommand.Command("all", "retry events all")
	)
	for i, a := range os.Args {
		if a == "-a" || a == "--all-keys" {
			os.Args[i] = "--all-keys=\000"
		}
	}
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case statusCommand.FullCommand():
		loadInit(*configFile)
		err := printStatus(*statusCommandSKey)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case detailCommand.FullCommand():
		loadInit(*configFile)
		err := printDetails(*detailCommandSKey, *detailCommandId)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case retryCommand.FullCommand():
		loadInit(*configFile)
		if *retryCommandAll == "\000" {
			if *retryCommandSKey != "" || *retryCommandId != "" {
				fmt.Println("-a option can't use with -k or -id.")
				os.Exit(0)
			}
			err := retry("", "")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			if *retryCommandSKey == "" && *retryCommandId == "" {
				fmt.Println("Please set Service Key or Id options")
				os.Exit(0)
			}
			err := retry(*retryCommandSKey, *retryCommandId)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}
	//putQueue()
}

func printStatus(skey string) (error){
	smap, err := getEvents(nil)
	if err != nil {
		return err
	}
	fmt.Printf("%-34s%10s%10s%10s\n", "Service Key", "Pending", "Success", "In Error")
	fmt.Printf("=================================================================\n")
	for _, k := range getSKeys(smap) {
		fmap := filterSKey(smap, k)
		if k == skey || skey == "" {
			fmt.Printf("%-34s%10d%10d%10d\n", k, len(fmap["in"]), len(fmap["succ"]), len(fmap["fail"]))
		}
	}
	return nil
}

func printDetails(skey string, id string) (error){
	smap, err := getEvents(nil)
	if err != nil {
		return err
	}
	for _, k := range getSKeys(smap) {
		fmap := filterSKey(smap, k)
		if k == skey || skey == "" {
			for s, ps := range fmap {
				fmt.Println("=====================")
				fmt.Printf("Status: %s\n", s)
				for _, p := range ps {
					if p.Id == id || id == "" {
						fmt.Printf("ID: %s\n", p.Id)
						j, err := json.MarshalIndent(p,"","  ")
						if err != nil {
							return err
						}
						fmt.Println(string(j))
					}
				}
			}
		}
	}
	return nil
}

func retry(skey string, id string) (error){
	count := 0
	smap, err := getEvents([]string{"fail"})
	if err != nil {
		return err
	}
	for _, k := range getSKeys(smap) {
		fmap := filterSKey(smap, k)
		if k == skey || skey == "" {
			for _, ps := range fmap {
				for _, p := range ps {
					if p.Id == id || id == "" {
						err = postRetry(p.Id)
						if err != nil {
							return err
						}
						count = count + 1
					}
				}
			}
		}
	}
	fmt.Printf("Retry %d events\n", count)
	return nil
}

func contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func filterSKey(smap map[string][]pdaclient.PdaEvent, skey string) (map[string][]pdaclient.PdaEvent) {
	status := []string{ "in", "fail", "succ" }
	fmap := make(map[string][]pdaclient.PdaEvent)
	for _, s := range status {
		var l []pdaclient.PdaEvent
		for _, p := range smap[s] {
			if p.Event.ServiceKey == skey {
				l = append(l, p)
			}
		} 
		fmap[s] = append(fmap[s], l...)
	}
	return fmap
}

func getEvents(status []string) (map[string][]pdaclient.PdaEvent, error) {
	if len(status) < 1 {
		status = []string{ "in", "fail", "succ" }
	}
	smap := make(map[string][]pdaclient.PdaEvent)
	for _, s := range status {
		l, err := getQueue(s)
		if err != nil {
			return smap, err
		}
		smap[s] = append(smap[s], l...)
	}
	return smap, nil
}

func getSKeys(smap map[string][]pdaclient.PdaEvent)([]string){
	var l []string
	for _, ps := range smap {
		for _, p := range ps { 
			if !contains(l, p.Event.ServiceKey) {
				l = append(l, p.Event.ServiceKey)
			}
		}
	}
	return l
}



func httpClient() (http.Client){
	return  http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
	}
}

func getQueue(prefix string) ([]pdaclient.PdaEvent, error) {
	var pdaList []pdaclient.PdaEvent
	httpc := httpClient()
	reqURL := "http://unix/search/" + prefix
	req, err := http.NewRequest("GET", reqURL , nil)
	if err != nil {
		return pdaList ,err
	}
	res, err := httpc.Do(req)
	if err != nil {
		return pdaList ,err
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return pdaList ,err
	}
	
	err = json.Unmarshal([]byte(b), &pdaList)
	if err != nil {
		return pdaList ,err
	}
	
	return pdaList, nil
}

func postRetry(id string) (error) {
	httpc := httpClient()
	reqURL := "http://unix/retry/" + id
	req, err := http.NewRequest("POST", reqURL , nil)
	if err != nil {
		return err
	}
	_, err = httpc.Do(req)
	if err != nil {
		return err
	}
	fmt.Printf("Retry ID: %s\n", id)
	return nil
}


