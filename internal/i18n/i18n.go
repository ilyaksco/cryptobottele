package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Translator struct {
	translations    map[string]map[string]string
	defaultLanguage string
}

func New(localesDir, defaultLanguage string) (*Translator, error) {
	t := &Translator{
		translations:    make(map[string]map[string]string),
		defaultLanguage: defaultLanguage,
	}

	files, err := os.ReadDir(localesDir)
	if err != nil {
		return nil, fmt.Errorf("could not read locales directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		langCode := strings.TrimSuffix(file.Name(), ".json")
		filePath := filepath.Join(localesDir, file.Name())

		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("could not read locale file %s: %w", file.Name(), err)
		}

		var messages map[string]string
		if err := json.Unmarshal(fileBytes, &messages); err != nil {
			return nil, fmt.Errorf("could not parse locale file %s: %w", file.Name(), err)
		}
		t.translations[langCode] = messages
	}

	return t, nil
}

func (t *Translator) Translate(langCode, key string, params map[string]string) string {
	messages, ok := t.translations[langCode]
	if !ok {
		messages = t.translations[t.defaultLanguage]
	}

	message, ok := messages[key]
	if !ok {
		return key
	}

	for k, v := range params {
		placeholder := "{" + k + "}"
		message = strings.ReplaceAll(message, placeholder, v)
	}

	return message
}