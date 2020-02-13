package main

import (
	"github.com/BurntSushi/toml"
)

type config struct {
	Main mainConfig `toml:"Main"`
}

type mainConfig struct {
	Socket                      string `toml:"socket"`
	SendIntervalSecs            int    `toml:"send_interval_secs"`
	CleanupIntervalSecs         int    `toml:"cleanup_interval_secs"`
	CleanupThresholdSecs        int    `toml:"cleanup_threshold_secs"`
	BackoffIntervalSecs         int    `toml:"backoff_interval_secs"`
	RetryLimitForPossibleErrors int    `toml:"retry_limit_for_possible_errors"`
	SourceAddress               string `toml:"source_address"`
}

func initConfig(path string) (config, error) {
	var conf config
	_, err := toml.DecodeFile(path, &conf)
	if err != nil {
		return conf, err
	}
	return conf, nil
}
