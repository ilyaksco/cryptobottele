package game

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ThemeLocale struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Template    string `yaml:"template"`
}

type Theme struct {
	ID    string      `yaml:"id"`
	Price int         `yaml:"price"`
	EN    ThemeLocale `yaml:"en"`
	IDLocale ThemeLocale `yaml:"id_locale"` // INI YANG DIPERBAIKI (sebelumnya 'ID')
}

type ThemeConfig struct {
	Themes []Theme `yaml:"themes"`
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