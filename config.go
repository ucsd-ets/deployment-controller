package main

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
)

const configFile = "/workspaces/deployment-controller/deployment-controller.yaml"

type KeyValue struct {
	Key   string `yaml:"Key"`
	Value string `yaml:"Value"`
}

// cookie data from config file
type Cookie struct {
	Expiration    string   `yaml:"Expiration"`
	CanaryPercent float32  `yaml:"CanaryPercent"`
	IfSuccessful  KeyValue `yaml:"IfSuccessful"`
	IfFail        KeyValue `yaml:"IfFail"`
}

type View struct {
	ShowSuccess bool `yaml:"ShowSuccess"`
	ShowFail    bool `yaml:"ShowFail"`
}

type Logging struct {
	Disable bool `yaml:"Disable"`
}

type App struct {
	Name       string  `yaml:"Name"`
	Disable    bool    `yaml:"Disable"`
	CookieInfo Cookie  `yaml:"CookieInfo"`
	View       View    `yaml:"View"`
	Logging    Logging `yaml:"Logging"`
}

type Config struct {
	Apps []App `yaml:"Apps"`
	Port int   `yaml:"Port"`
}

func ReadConfig() (Config, error) {
	// set path, use /workspaces/.. if unspecified
	configPath := os.Getenv("APP_CONFIG_PATH")
	if configPath == "" {
		configPath = "/workspaces/deployment-controller/deployment-controller.yaml"
	}
	config := Config{}
	configYaml, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(configYaml, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func UpdateConfig(app App) error {
	config, err := ReadConfig()
	if err != nil {
		return err
	}

	appFound := false
	for i, configapp := range config.Apps {
		if app.Name == configapp.Name {
			configapp = app
			config.Apps[i] = configapp
			appFound = true
		}
	}

	if !appFound {
		return errors.New("could not find app")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(configFile, data, 0)
	if err != nil {
		return err
	}
	return nil
}
