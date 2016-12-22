package kumo

import (
	"encoding/json"
	"io"
	"io/ioutil"
)

type Config struct {
	Debug       bool
	DebugFile   string
	Quiet       bool
	Connections int
	Host        string
	Username    string
	Password    string
	Port        int
	Temp        string
	Download    string
	SSL         bool
	Filters     []string
}

func GetConfig(f io.Reader) (*Config, error) {
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	config := new(Config)
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
