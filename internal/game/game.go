package game

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var superscriptMap = map[rune]string{
	'0': "⁰", '1': "¹", '2': "²", '3': "³", '4': "⁴",
	'5': "⁵", '6': "⁶", '7': "⁷", '8': "⁸", '9': "⁹",
}

type Puzzle struct {
	Word         string
	Display      string
	Solution     string
	HiddenCount  int
	MessageID    int
}

type Service struct {
	config *Config
	random *rand.Rand
}

func NewService(config *Config) *Service {
	source := rand.NewSource(time.Now().UnixNano())
	return &Service{
		config: config,
		random: rand.New(source),
	}
}

func (s *Service) GeneratePuzzle() *Puzzle {
	word := s.config.Words[s.random.Intn(len(s.config.Words))]
	word = strings.ToUpper(word)

	var displayBuilder strings.Builder
	var solutionBuilder strings.Builder
	
	letterIndices := []int{}
	for i, char := range word {
		if unicode.IsLetter(char) {
			letterIndices = append(letterIndices, i)
		}
	}

	s.random.Shuffle(len(letterIndices), func(i, j int) {
		letterIndices[i], letterIndices[j] = letterIndices[j], letterIndices[i]
	})

	hideCount := (len(letterIndices) * 4) / 10 
	if hideCount == 0 && len(letterIndices) > 1 {
		hideCount = 1
	}

	hideSet := make(map[int]bool)
	for i := 0; i < hideCount; i++ {
		hideSet[letterIndices[i]] = true
	}

	for i, char := range word {
		if !unicode.IsLetter(char) && char != ' ' {
			continue
		}

		if char == ' ' {
			displayBuilder.WriteString("  ")
			continue
		}

		cryptoVal := int(char-'A') + 1 + s.config.Shift
		superScript := toSuperscript(cryptoVal)

		if hideSet[i] {
			displayBuilder.WriteString("(_" + superScript + ")")
			solutionBuilder.WriteRune(char)
		} else {
			displayBuilder.WriteString("(" + string(char) + superScript + ")")
		}
	}

	return &Puzzle{
		Word:        word,
		Display:     displayBuilder.String(),
		Solution:    solutionBuilder.String(),
		HiddenCount: hideCount,
	}
}

func (s *Service) CheckAnswer(solution, guess string) bool {
	return strings.ToUpper(guess) == solution
}

func toSuperscript(n int) string {
	s := strconv.Itoa(n)
	var result strings.Builder
	for _, r := range s {
		result.WriteString(superscriptMap[r])
	}
	return result.String()
}