package game

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type PuzzleConfig struct {
	Text  string      `yaml:"text"`
	Shift interface{} `yaml:"shift"`
}

type Config struct {
	Puzzles []PuzzleConfig `yaml:"puzzles"`
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

	if len(config.Puzzles) == 0 {
		return nil, fmt.Errorf("no puzzles found in game config")
	}

	return &config, nil
}