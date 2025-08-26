package bot

import (
	"log"
	"strconv"
	"strings"

	"cryptowordgamebot/internal/config"
	"cryptowordgamebot/internal/game"
	"cryptowordgamebot/internal/i18n"
	"cryptowordgamebot/internal/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotHandler struct {
	bot           *tgbotapi.BotAPI
	translator    *i18n.Translator
	config        *config.Config
	storage       *storage.Storage
	gameSvc       *game.Service
	activePuzzles map[int64]*game.Puzzle
}

func NewBotHandler(bot *tgbotapi.BotAPI, trans *i18n.Translator, cfg *config.Config, store *storage.Storage, gameSvc *game.Service) *BotHandler {
	return &BotHandler{
		bot:           bot,
		translator:    trans,
		config:        cfg,
		storage:       store,
		gameSvc:       gameSvc,
		activePuzzles: make(map[int64]*game.Puzzle),
	}
}

func (h *BotHandler) HandleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	user, err := h.ensureUserExists(update.Message)
	if err != nil {
		log.Printf("Failed to ensure user exists: %v", err)
		return
	}
	if update.Message.IsCommand() {
		h.handleCommand(update.Message, user)
		return
	}
	isPrivate := update.Message.Chat.IsPrivate()
	puzzle, isActive := h.activePuzzles[update.Message.Chat.ID]
	if isPrivate && isActive {
		h.handleGuess(update.Message, user, puzzle)
		return
	}
	if !isPrivate && isActive && update.Message.ReplyToMessage != nil && puzzle.MessageID == update.Message.ReplyToMessage.MessageID {
		h.handleGuess(update.Message, user, puzzle)
		return
	}
}

func (h *BotHandler) handleGuess(message *tgbotapi.Message, user *storage.User, puzzle *game.Puzzle) {
	result := h.gameSvc.CheckAnswer(puzzle.RemainingSolution, message.Text)

	if !result.IsCorrect && !result.IsPartial {
		responseText := h.translator.Translate(user.LanguageCode, "wrong_answer", nil)
		h.sendMessage(message.Chat.ID, responseText, "")
		return
	}

	puzzle.UpdateState(result.CorrectlyGuessedChars)
	newPuzzleText := "`" + puzzle.RenderDisplay() + "`"
	h.editMessage(message.Chat.ID, puzzle.MessageID, newPuzzleText, tgbotapi.ModeMarkdownV2)

	// Periksa status puzzle SETELAH diupdate
	if puzzle.RemainingSolution == "" {
		delete(h.activePuzzles, message.Chat.ID)
		points := 10
		newScore, err := h.storage.IncreaseUserScore(user.ID, points)
		if err != nil {
			log.Printf("Failed to increase score for user %d: %v", user.ID, err)
			return
		}
		params := map[string]string{
			"points":      strconv.Itoa(points),
			"total_score": strconv.FormatInt(newScore, 10),
		}
		responseText := h.translator.Translate(user.LanguageCode, "correct_answer", params)
		h.sendMessage(message.Chat.ID, responseText, tgbotapi.ModeHTML)
	} else {
		params := map[string]string{"guessed_chars": result.CorrectlyGuessedChars}
		responseText := h.translator.Translate(user.LanguageCode, "partial_correct", params)
		h.sendMessage(message.Chat.ID, responseText, "")
	}
}


// ... Sisa file (ensureUserExists, handleCommand, dll) tetap sama
func (h *BotHandler) ensureUserExists(message *tgbotapi.Message) (*storage.User, error) {
	user, err := h.storage.GetUser(message.From.ID)
	if err != nil {
		user = &storage.User{
			ID:           message.From.ID,
			LanguageCode: h.config.DefaultLanguage,
		}
	}
	user.FirstName = message.From.FirstName
	user.LastName = message.From.LastName
	user.Username = message.From.UserName
	if err := h.storage.UpsertUser(*user); err != nil {
		return nil, err
	}
	return user, nil
}
func (h *BotHandler) handleCommand(message *tgbotapi.Message, user *storage.User) {
	switch message.Command() {
	case "start":
		h.handleStartCommand(message, user)
	case "lang":
		h.handleLangCommand(message, user)
	case "crypto":
		h.handleCryptoCommand(message, user)
	case "score":
		h.handleScoreCommand(message, user)
	case "profile":
		h.handleProfileCommand(message, user)
	case "leaderboard":
		h.handleLeaderboardCommand(message, user)
	case "surrender", "menyerah":
		h.handleSurrenderCommand(message, user)
	}
}

func (h *BotHandler) handleSurrenderCommand(message *tgbotapi.Message, user *storage.User) {
	puzzle, isActive := h.activePuzzles[message.Chat.ID]
	if !isActive {
		responseText := h.translator.Translate(user.LanguageCode, "no_active_puzzle", nil)
		h.sendMessage(message.Chat.ID, responseText, "")
		return
	}

	puzzle.RevealAll()
	// 2. Render puzzle yang sudah lengkap
	finalText := "`" + puzzle.RenderDisplay() + "`"
	// 3. EDIT pesan puzzle yang asli
	h.editMessage(message.Chat.ID, puzzle.MessageID, finalText, tgbotapi.ModeMarkdownV2)

	// 4. Hapus sesi setelah semuanya selesai
	delete(h.activePuzzles, message.Chat.ID)

	params := map[string]string{"answer": puzzle.Solution}
	responseText := h.translator.Translate(user.LanguageCode, "surrender_message", params)
	h.sendMessage(message.Chat.ID, responseText, tgbotapi.ModeHTML)
	
}

func (h *BotHandler) handleCryptoCommand(message *tgbotapi.Message, user *storage.User) {
	if !message.Chat.IsPrivate() {
		if _, ok := h.activePuzzles[message.Chat.ID]; ok {
			responseText := h.translator.Translate(user.LanguageCode, "puzzle_in_progress", nil)
			h.sendMessage(message.Chat.ID, responseText, "")
			return
		}
	}
	puzzle := h.gameSvc.GeneratePuzzle()
	params := map[string]string{"count": strconv.Itoa(len(puzzle.Solution))}
	introText := h.translator.Translate(user.LanguageCode, "new_puzzle", params)
	h.sendMessage(message.Chat.ID, introText, "")

	puzzleText := "`" + puzzle.RenderDisplay() + "`"
	sentMsg, err := h.sendMessage(message.Chat.ID, puzzleText, tgbotapi.ModeMarkdownV2)
	if err != nil {
		log.Printf("Failed to send puzzle message: %v", err)
		return
	}
	puzzle.MessageID = sentMsg.MessageID
	h.activePuzzles[message.Chat.ID] = puzzle
}
func (h *BotHandler) editMessage(chatID int64, messageID int, text string, parseMode string) {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if parseMode != "" {
		msg.ParseMode = parseMode
	}
	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to edit message: %v", err)
	}
}
func (h *BotHandler) handleStartCommand(message *tgbotapi.Message, user *storage.User) {
	params := map[string]string{"name": message.From.FirstName}
	responseText := h.translator.Translate(user.LanguageCode, "welcome", params)
	h.sendMessage(message.Chat.ID, responseText, "")
}
func (h *BotHandler) handleLangCommand(message *tgbotapi.Message, user *storage.User) {
	args := message.CommandArguments()
	langCode := strings.ToLower(strings.TrimSpace(args))
	if langCode != "en" && langCode != "id" {
		responseText := h.translator.Translate(user.LanguageCode, "lang_usage", nil)
		h.sendMessage(message.Chat.ID, responseText, "")
		return
	}
	err := h.storage.UpdateUserLanguage(user.ID, langCode)
	if err != nil {
		log.Printf("Failed to update user language: %v", err)
		responseText := h.translator.Translate(user.LanguageCode, "lang_change_failed", nil)
		h.sendMessage(message.Chat.ID, responseText, "")
		return
	}
	responseText := h.translator.Translate(langCode, "lang_changed", nil)
	h.sendMessage(message.Chat.ID, responseText, "")
}
func (h *BotHandler) handleScoreCommand(message *tgbotapi.Message, user *storage.User) {
	params := map[string]string{"score": strconv.FormatInt(user.Score, 10)}
	responseText := h.translator.Translate(user.LanguageCode, "user_score", params)
	h.sendMessage(message.Chat.ID, responseText, tgbotapi.ModeHTML)
}
func (h *BotHandler) handleProfileCommand(message *tgbotapi.Message, user *storage.User) {
	params := map[string]string{
		"name":  user.FirstName,
		"score": strconv.FormatInt(user.Score, 10),
	}
	responseText := h.translator.Translate(user.LanguageCode, "profile_info", params)
	h.sendMessage(message.Chat.ID, responseText, tgbotapi.ModeHTML)
}
func (h *BotHandler) handleLeaderboardCommand(message *tgbotapi.Message, user *storage.User) {
	topUsers, err := h.storage.GetTopUsers(10)
	if err != nil {
		log.Printf("Failed to get top users for leaderboard: %v", err)
		return
	}
	var leaderboardBuilder strings.Builder
	title := h.translator.Translate(user.LanguageCode, "leaderboard_title", nil)
	leaderboardBuilder.WriteString(title)
	for i, player := range topUsers {
		params := map[string]string{
			"rank":  strconv.Itoa(i + 1),
			"name":  player.FirstName,
			"score": strconv.FormatInt(player.Score, 10),
		}
		entry := h.translator.Translate(user.LanguageCode, "leaderboard_entry", params)
		leaderboardBuilder.WriteString(entry)
	}
	h.sendMessage(message.Chat.ID, leaderboardBuilder.String(), tgbotapi.ModeHTML)
}
func (h *BotHandler) sendMessage(chatID int64, text string, parseMode string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	if parseMode != "" {
		msg.ParseMode = parseMode
	}
	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
	return sentMsg, err
}