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

type DifficultyLevel struct {
	Points         int            `yaml:"points"`
	HidePercentage int            `yaml:"hide_percentage"`
	Puzzles        []PuzzleConfig `yaml:"puzzles"`
}

type Config struct {
	Difficulties map[string]DifficultyLevel `yaml:"difficulties"`
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

	if len(config.Difficulties) == 0 {
		return nil, fmt.Errorf("no difficulties found in game config")
	}

	return &config, nil
}