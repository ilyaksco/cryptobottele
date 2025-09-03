package game

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
	"unicode"
	"fmt"
)

var superscriptMap = map[rune]string{
	'0': "⁰", '1': "¹", '2': "²", '3': "³", '4': "⁴",
	'5': "⁵", '6': "⁶", '7': "⁷", '8': "⁸", '9': "⁹",
}

type PuzzleChar struct {
	Char      rune
	IsHidden  bool
	IsGuessed bool
	Value     int
}

type Puzzle struct {
	Chars             []*PuzzleChar
	Solution          string
	RemainingSolution string
	MessageID         int
	Points            int
}

type Service struct {
	config *Config
	random *rand.Rand
}

type CheckResult struct {
	IsCorrect           bool
	IsPartial           bool
	CorrectlyGuessedChars string
}

func NewService(config *Config) *Service {
	source := rand.NewSource(time.Now().UnixNano())
	return &Service{
		config: config,
		random: rand.New(source),
	}
}

func (s *Service) GeneratePuzzle(difficulty string) (*Puzzle, error) {
	level, ok := s.config.Difficulties[difficulty]
	if !ok {
		level, ok = s.config.Difficulties["easy"]
		if !ok {
			return nil, fmt.Errorf("easy difficulty level not found in config")
		}
	}

	if len(level.Puzzles) == 0 {
		return nil, fmt.Errorf("no puzzles found for difficulty: %s", difficulty)
	}

	puzzleConfig := level.Puzzles[s.random.Intn(len(level.Puzzles))]
	word := strings.ToUpper(puzzleConfig.Text)

	var finalShift int
	switch v := puzzleConfig.Shift.(type) {
	case int:
		finalShift = v
	case string:
		if v == "random" {
			finalShift = s.random.Intn(10) + 1
		}
	default:
		finalShift = 0
	}

	var puzzleChars []*PuzzleChar
	var solutionBuilder strings.Builder
	var letterIndices []int

	for i, char := range word {
		if unicode.IsLetter(char) {
			letterIndices = append(letterIndices, i)
		}
	}

	s.random.Shuffle(len(letterIndices), func(i, j int) {
		letterIndices[i], letterIndices[j] = letterIndices[j], letterIndices[i]
	})

	hideCount := (len(letterIndices) * level.HidePercentage) / 100
	if hideCount == 0 && len(letterIndices) > 1 {
		hideCount = 1
	}

	hideSet := make(map[int]bool)
	for i := 0; i < hideCount; i++ {
		hideSet[letterIndices[i]] = true
	}

	for _, char := range word {
		pc := &PuzzleChar{Char: char}
		if unicode.IsLetter(char) {
			pc.Value = int(char-'A') + 1 + finalShift
		}
		puzzleChars = append(puzzleChars, pc)
	}

	for i, pc := range puzzleChars {
		if hideSet[i] {
			pc.IsHidden = true
			solutionBuilder.WriteRune(pc.Char)
		}
	}

	solution := solutionBuilder.String()
	return &Puzzle{
		Chars:             puzzleChars,
		Solution:          solution,
		RemainingSolution: solution,
		Points:            level.Points,
	}, nil
}

func (p *Puzzle) RevealAll() {
	for _, pc := range p.Chars {
		if pc.IsHidden {
			pc.IsGuessed = true
		}
	}
}

func (p *Puzzle) RenderDisplay() string {
	var displayBuilder strings.Builder
	for _, pc := range p.Chars {
		if pc.Char == ' ' {
			displayBuilder.WriteString("\n")
			continue
		}
		if !unicode.IsLetter(pc.Char) {
			continue
		}

		superScript := toSuperscript(pc.Value)
		if pc.IsHidden && !pc.IsGuessed {
			displayBuilder.WriteString("(_" + superScript + ")")
		} else {
			displayBuilder.WriteString("(" + string(pc.Char) + superScript + ")")
		}
	}
	return displayBuilder.String()
}

func (p *Puzzle) UpdateState(guessedChars string) {
	guessedMap := make(map[rune]int)
	for _, r := range guessedChars {
		guessedMap[r]++
	}

	var newRemainingSolution strings.Builder
	for _, r := range p.RemainingSolution {
		if count, ok := guessedMap[r]; ok && count > 0 {
			guessedMap[r]--
		} else {
			newRemainingSolution.WriteRune(r)
		}
	}
	p.RemainingSolution = newRemainingSolution.String()

	guessedMap = make(map[rune]int)
	for _, r := range guessedChars {
		guessedMap[r]++
	}

	for _, pc := range p.Chars {
		if pc.IsHidden && !pc.IsGuessed {
			if count, ok := guessedMap[pc.Char]; ok && count > 0 {
				pc.IsGuessed = true
				guessedMap[pc.Char]--
			}
		}
	}
}

func (s *Service) CheckAnswer(remainingSolution, guess string) *CheckResult {
	guess = strings.ToUpper(guess)
	result := &CheckResult{}

	tempSolution := remainingSolution
	for _, char := range guess {
		if !strings.ContainsRune(tempSolution, char) {
			return result // Kembalikan hasil kosong (salah total)
		}
	}

	if guess == remainingSolution {
		result.IsCorrect = true
		result.CorrectlyGuessedChars = guess // Penambahan penting di sini
		return result
	}

	var guessedCharsBuilder strings.Builder
	solutionMap := make(map[rune]int)
	for _, r := range remainingSolution {
		solutionMap[r]++
	}

	for _, r := range guess {
		if count, ok := solutionMap[r]; ok && count > 0 {
			guessedCharsBuilder.WriteRune(r)
			solutionMap[r]--
		}
	}

	guessedStr := guessedCharsBuilder.String()
	if len(guessedStr) > 0 {
		result.IsPartial = true
		result.CorrectlyGuessedChars = guessedStr
	}

	return result
}

func toSuperscript(n int) string {
	s := strconv.Itoa(n)
	var result strings.Builder
	for _, r := range s {
		result.WriteString(superscriptMap[r])
	}
	return result.String()
}

// File: internal/game/game.go

// ▼▼▼ ADD THIS NEW FUNCTION AT THE END OF THE FILE ▼▼▼
func (p *Puzzle) RevealRandomChar() (revealedChar rune, success bool) {
	var hiddenIndices []int
	for i, pc := range p.Chars {
		if pc.IsHidden && !pc.IsGuessed {
			hiddenIndices = append(hiddenIndices, i)
		}
	}

	if len(hiddenIndices) == 0 {
		return 0, false
	}

	rand.Shuffle(len(hiddenIndices), func(i, j int) {
		hiddenIndices[i], hiddenIndices[j] = hiddenIndices[j], hiddenIndices[i]
	})
	
	revealIndex := hiddenIndices[0]
	revealedChar = p.Chars[revealIndex].Char
	
	p.UpdateState(string(revealedChar))

	return revealedChar, true
}
// ▲▲▲ ADD THIS NEW FUNCTION AT THE END OF THE FILE ▲▲▲