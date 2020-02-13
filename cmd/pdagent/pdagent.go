package main

import (
	"os"
	"net"
	"os/signal"
	"syscall"
	"sync"
	"context"
	"fmt"
	"time"
        "strings"
	"github.com/zenazn/goji/graceful"
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/peterbourgon/diskv"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"

	"github.com/m-masataka/go-pdagent/pkg/pdaclient"
)

func init() {
}

func initLogger(logLevel string) log.Logger {
	w := log.NewSyncWriter(os.Stdout)
        logger := log.NewJSONLogger(w)
	format := log.TimestampFormat(
		func() time.Time { return time.Now().UTC() },
		time.RFC3339Nano,
	)
	if logLevel == "debug" {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}
	return log.With(logger, "timestamp", format, "caller", log.DefaultCaller)
}


func advancedTransform(key string) *diskv.PathKey {
	path := strings.Split(key, "/")
	last := len(path) - 1
	return &diskv.PathKey{
		Path:     path[:last],
		FileName: path[last],
	}
}

func inverseTransform(pathKey *diskv.PathKey) (key string) {
	return strings.Join( append(pathKey.Path, pathKey.FileName) , "/")
}


type Item struct {
	Name string
	Id   int
}

func ItemBuilder() interface{} {
	return &pdaclient.PdaEvent{}
}


var wg sync.WaitGroup

func main() {
	os.Exit(run())
}

func run() int {
	var (
		dataDir    = kingpin.Flag("storage.path", "Base path for data storage.").Default("data/").String()
		configPath = kingpin.Flag("config", "Path for configration file.").Default("config/pdagent.conf").String()
		logLevel   = kingpin.Flag("log.level", "Set log level debug or info.").Default("info").String()
	)
	kingpin.CommandLine.GetFlag("help").Short('h')
	kingpin.Parse()

        pid := os.Getpid()
	logger := initLogger(*logLevel)
	level.Info(logger).Log("msg", "pdagentd starting!", "pid", pid)

	// Load Configuration file
	conf, err := initConfig(*configPath)
	if err != nil {
		level.Error(logger).Log("msg", "Unable to load config file", "err", err)
		return 1
	}
	interval := conf.Main.SendIntervalSecs
	backoffInterval := conf.Main.BackoffIntervalSecs
	retryLimit := conf.Main.RetryLimitForPossibleErrors
	cleanupInterval := conf.Main.CleanupIntervalSecs
	cleanupTh := conf.Main.CleanupThresholdSecs
	sockPath   := conf.Main.Socket

	var term = make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	if err = os.MkdirAll(*dataDir, 0777); err != nil {
		level.Error(logger).Log("msg", "Unable to create data directory", "err", err)
		return 1
	}

	d := diskv.New(diskv.Options{
		BasePath:     *dataDir,
		AdvancedTransform: advancedTransform, 
		InverseTransform: inverseTransform,
		CacheSizeMax: 1024 * 1024,
	})

	defer os.Remove(sockPath)
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return 1
	}

	eqHandler := &enqueueHandler{ logger, d }
	searchHandler := &searchHandler{ logger, d }
	retryHandler := &retryHandler{ logger, d }
	r := mux.NewRouter()
	r.Handle("/enque", eqHandler)
	r.Handle("/search/{prefix}", searchHandler)
	r.Handle("/retry/{id}", retryHandler)


	go func() {
		if err := graceful.Serve(listener, r); err != nil {
			level.Error(logger).Log("err", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	wg.Add(1)
	go backoff(ctx, d, backoffInterval, retryLimit, logger)
	wg.Add(1)
	go cleanup(ctx, d, cleanupInterval, cleanupTh, logger)
	wg.Add(1)
	go notifier(ctx, d, interval, logger)

	for {
		select {
			case <-term:
				level.Info(logger).Log("msg", "shutdown!", "pid", pid)
				graceful.Shutdown()
				listener.Close()
				return 0
		}
	}
	fmt.Println("FIN")
	return 0
}
