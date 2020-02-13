package main

import (
	"bytes"
	"os"
	"fmt"
	"net"
	"net/http"
	"context"
	"strings"
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
		configFile  = kingpin.Flag("config", "Path for config file.").Default("config/pdagent.conf").String()
		serviceKey  = kingpin.Arg("service-key", "service key").Required().String()
		messageType = kingpin.Arg("message-type", "message type").Required().String()
		details     = kingpin.Arg("details", "details").Required().String()
	)
	kingpin.CommandLine.GetFlag("help").Short('h')
	kingpin.Parse()
	loadInit(*configFile)
	err := putQueue(*serviceKey, *messageType, *details)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		os.Exit(1)
	}
}

func putQueue(serviceKey string, messageType string, details string) (error){
	mapd, err := parseZabbixBody(details)
	if err != nil {
		return err
	}
	httpc := httpClient()
	incidentKey := mapd["id"] + mapd["hostname"]
	description := fmt.Sprintf("%s : %s for %s", mapd["name"], mapd["status"], mapd["hostname"])
	j, err := pdaclient.QueueEvent("trigger", serviceKey, incidentKey, description, "", "", mapd, "a2", "pd-zabbix")
	req, err := http.NewRequest("POST", "http://unix/enque", bytes.NewBuffer(j.EventToBytes()))
	if err != nil {
		return err
	}

	res, err := httpc.Do(req)
	if err != nil {
		return err
	}

	// header
	fmt.Printf("[status] %d\n", res.StatusCode)
	for k, v := range res.Header {
		fmt.Print("[header] " + k)
		fmt.Println(": " + strings.Join(v, ","))
	}
	return nil
}

func parseZabbixBody(body string) (map[string]string, error){
	b := make(map[string]string)
	for _, s := range strings.Split(body, "\n") {
		d := strings.SplitN(s, ":", 2)
		if len(d) != 2 {
			return b, fmt.Errorf("Parse Error")
		}
		b[d[0]] = d[1]
	}
	return b, nil
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
