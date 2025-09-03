package game

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ItemLocale struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Template    string `yaml:"template,omitempty"`
}

type MarketItem struct {
	ID       string     `yaml:"id"`
	Price    int        `yaml:"price"`
	EN       ItemLocale `yaml:"en"`
	IDLocale ItemLocale `yaml:"id_locale"`
}

type ThemeConfig struct {
	Themes []MarketItem `yaml:"themes"`
}

type PowerupConfig struct {
	Powerups []MarketItem `yaml:"powerups"`
}

func LoadThemes(filePath string) (*ThemeConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read themes config file: %w", err)
	}

	var config ThemeConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("could not parse themes config file: %w", err)
	}

	return &config, nil
}

func LoadPowerups(filePath string) (*PowerupConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read powerups config file: %w", err)
	}

	var config PowerupConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("could not parse powerups config file: %w", err)
	}

	return &config, nil
}