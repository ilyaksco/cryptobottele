package game

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Shift int      `yaml:"shift"`
	Words []string `yaml:"words"`
}

func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read game config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("could not parse game config file: %w", err)
	}

	if len(config.Words) == 0 {
		return nil, fmt.Errorf("no words found in game config")
	}

	return &config, nil
}