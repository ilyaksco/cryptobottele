package bot

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"fmt"

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
	themeConfig   *game.ThemeConfig 
	powerupConfig *game.PowerupConfig
	activePuzzles map[int64]*game.Puzzle
	mu            sync.Mutex

	
}

func NewBotHandler(bot *tgbotapi.BotAPI, trans *i18n.Translator, cfg *config.Config, store *storage.Storage, gameSvc *game.Service, themeCfg *game.ThemeConfig, powerupCfg *game.PowerupConfig) *BotHandler {
	return &BotHandler{
		bot:           bot,
		translator:    trans,
		config:        cfg,
		storage:       store,
		gameSvc:       gameSvc,
		themeConfig:   themeCfg,
		powerupConfig: powerupCfg,
		activePuzzles: make(map[int64]*game.Puzzle),
		
	}
}

// vvv AWAL PERUBAHAN vvv
func (h *BotHandler) HandleUpdate(update tgbotapi.Update) {
	var fromUser *tgbotapi.User
	if update.Message != nil {
		fromUser = update.Message.From
	} else if update.CallbackQuery != nil {
		fromUser = update.CallbackQuery.From
	} else {
		return
	}

	user, err := h.ensureUserExists(fromUser)
	if err != nil {
		log.Printf("Failed to ensure user exists: %v", err)
		return
	}

	if update.Message != nil {
		if update.Message.IsCommand() {
			h.handleCommand(update.Message, user)
			return
		}

		h.mu.Lock()
		puzzle, isActive := h.activePuzzles[update.Message.Chat.ID]
		h.mu.Unlock()

		if isActive {
			isPrivate := update.Message.Chat.IsPrivate()
			if isPrivate || (update.Message.ReplyToMessage != nil && puzzle.MessageID == update.Message.ReplyToMessage.MessageID) {
				h.handleGuess(update.Message, user, puzzle)
				return
			}
		}
	} else if update.CallbackQuery != nil {
		h.handleCallbackQuery(update.CallbackQuery, user)
	}
}
// ^^^ AKHIR PERUBAHAN ^^^


// vvv GANTI DENGAN FUNGSI INI vvv
func (h *BotHandler) handleCallbackQuery(query *tgbotapi.CallbackQuery, user *storage.User) {


	if strings.HasPrefix(query.Data, "market_") || strings.HasPrefix(query.Data, "powerup_") {
		h.handleMarketCallback(query, user)
		return
	}

    var sendNewMessage bool
    var text string
    var markup tgbotapi.InlineKeyboardMarkup


	

    switch query.Data {
    case "play_again":
        sendNewMessage = true
        h.handleCryptoCommand(query.Message, user)
    case "help_howtoplay":
        text = h.translator.Translate(user.LanguageCode, "help_text_howtoplay", nil)
        markup = h.buildHelpKeyboard(user.LanguageCode, "back_only")
    case "help_whatiscrypto":
        text = h.translator.Translate(user.LanguageCode, "help_text_whatiscrypto", nil)
        markup = h.buildHelpKeyboard(user.LanguageCode, "back_only")
    case "help_commands":
        text = h.translator.Translate(user.LanguageCode, "help_text_commands", nil)
        markup = h.buildHelpKeyboard(user.LanguageCode, "back_only")
    case "help_main":
        fallthrough
    default:
        text = h.translator.Translate(user.LanguageCode, "help_intro", nil)
        markup = h.buildHelpKeyboard(user.LanguageCode, "main")
    }

    if !sendNewMessage {
        msg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, text)
        msg.ParseMode = tgbotapi.ModeHTML
        msg.ReplyMarkup = &markup
        h.bot.Request(msg)
    }

    h.bot.Request(tgbotapi.NewCallback(query.ID, ""))
}

// ▼▼▼ FUNGSI-FUNGSI BARU DITAMBAHKAN ▼▼▼
func (h *BotHandler) handleMarketCallback(query *tgbotapi.CallbackQuery, user *storage.User) {
	h.bot.Request(tgbotapi.NewCallback(query.ID, ""))

	dataParts := strings.SplitN(query.Data, "_", 3)
	command := dataParts[0]
	subcommand := dataParts[1]
	var args []string
	if len(dataParts) > 2 {
		args = append(args, dataParts[2])
	}

	log.Printf("Handling market callback: command=%s, subcommand=%s, args=%v", command, subcommand, args)

	currentUser, err := h.storage.GetUser(user.ID)
	if err != nil {
		log.Printf("Error getting user for market callback: %v", err)
		return
	}

	switch command {
	case "market":
		h.handleMarketNavigation(query, currentUser, subcommand, args)
	case "powerup":
		h.handlePowerupActions(query, currentUser, subcommand, args)
	}
}

func (h *BotHandler) handleMarketNavigation(query *tgbotapi.CallbackQuery, user *storage.User, page string, args []string) {
	switch page {
	case "themes":
		text := h.translator.Translate(user.LanguageCode, "market_intro", nil)
		var keyboardRows [][]tgbotapi.InlineKeyboardButton
		for _, theme := range h.themeConfig.Themes {
			if theme.Price > 0 {
				var themeName string
				if user.LanguageCode == "id" {
					themeName = theme.IDLocale.Name
				} else {
					themeName = theme.EN.Name
				}
				button := tgbotapi.NewInlineKeyboardButtonData(themeName, "market_view_"+theme.ID)
				keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(button))
			}
		}
		backButtonText := h.translator.Translate(user.LanguageCode, "market_back_to_main", nil)
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(backButtonText, "market_main")))
		markup := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
		msg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, text)
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = &markup
		h.bot.Request(msg)

	case "powerups":
		text := h.translator.Translate(user.LanguageCode, "market_powerups_intro", nil)
		var keyboardRows [][]tgbotapi.InlineKeyboardButton
		for _, powerup := range h.powerupConfig.Powerups {
			var powerupName string
			if user.LanguageCode == "id" {
				powerupName = powerup.IDLocale.Name
			} else {
				powerupName = powerup.EN.Name
			}
			buttonText := fmt.Sprintf("%s (%d Poin)", powerupName, powerup.Price)
			button := tgbotapi.NewInlineKeyboardButtonData(buttonText, "market_buypowerup_"+powerup.ID)
			keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(button))
		}
		backButtonText := h.translator.Translate(user.LanguageCode, "market_back_to_main", nil)
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(backButtonText, "market_main")))
		markup := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
		msg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, text)
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = &markup
		h.bot.Request(msg)

	case "view":
		themeID := args[0]
		var selectedTheme *game.MarketItem
		for i := range h.themeConfig.Themes {
			if h.themeConfig.Themes[i].ID == themeID {
				selectedTheme = &h.themeConfig.Themes[i]
				break
			}
		}
		if selectedTheme == nil {
			return
		}
		var localeData game.ItemLocale
		if user.LanguageCode == "id" {
			localeData = selectedTheme.IDLocale
		} else {
			localeData = selectedTheme.EN
		}
		themeName := localeData.Name
		themeDesc := localeData.Description
		previewParams := map[string]string{"name": user.FirstName, "score": strconv.FormatInt(user.Score, 10)}
		profilePreview := localeData.Template
		for k, v := range previewParams {
			profilePreview = strings.ReplaceAll(profilePreview, "{"+k+"}", v)
		}
		var previewTextBuilder strings.Builder
		previewTextBuilder.WriteString(fmt.Sprintf("<b>%s</b>\n", themeName))
		previewTextBuilder.WriteString(fmt.Sprintf("<i>%s</i>\n\n", themeDesc))
		previewTextBuilder.WriteString("<b>Pratinjau:</b>\n")
		previewTextBuilder.WriteString(profilePreview)
		var buttons []tgbotapi.InlineKeyboardButton
		buyButtonText := h.translator.Translate(user.LanguageCode, "market_button_buy", map[string]string{"price": strconv.Itoa(selectedTheme.Price)})
		if user.ProfileTheme == selectedTheme.ID {
			buyButtonText = h.translator.Translate(user.LanguageCode, "market_preview_owned", nil)
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(buyButtonText, "noop"))
		} else {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(buyButtonText, "market_buytheme_"+selectedTheme.ID))
		}
		backButtonText := h.translator.Translate(user.LanguageCode, "market_button_back", nil)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(backButtonText, "market_themes"))
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttons...))
		msg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, previewTextBuilder.String())
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = &markup
		h.bot.Request(msg)

	case "buytheme":
		themeID := args[0]
		var selectedTheme *game.MarketItem
		for i := range h.themeConfig.Themes {
			if h.themeConfig.Themes[i].ID == themeID {
				selectedTheme = &h.themeConfig.Themes[i]
				break
			}
		}
		if selectedTheme == nil {
			return
		}
		if user.ProfileTheme == selectedTheme.ID {
			responseText := h.translator.Translate(user.LanguageCode, "market_already_owned", nil)
			h.sendMessage(query.Message.Chat.ID, responseText, tgbotapi.ModeHTML)
			return
		}
		cost := int64(selectedTheme.Price)
		if user.Score < cost {
			responseText := h.translator.Translate(user.LanguageCode, "market_not_enough_points", nil)
			h.sendMessage(query.Message.Chat.ID, responseText, tgbotapi.ModeHTML)
			return
		}
		h.storage.IncreaseUserScore(user.ID, -int(cost))
		h.storage.UpdateUserProfileTheme(user.ID, selectedTheme.ID)
		var themeName string
		if user.LanguageCode == "id" {
			themeName = selectedTheme.IDLocale.Name
		} else {
			themeName = selectedTheme.EN.Name
		}
		responseText := h.translator.Translate(user.LanguageCode, "market_purchase_success", map[string]string{"item": themeName})
		h.sendMessage(query.Message.Chat.ID, responseText, tgbotapi.ModeHTML)
		deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)
		h.bot.Request(deleteMsg)

	case "buypowerup":
		powerupID := args[0]
		var selectedPowerup *game.MarketItem
		for i := range h.powerupConfig.Powerups {
			if h.powerupConfig.Powerups[i].ID == powerupID {
				selectedPowerup = &h.powerupConfig.Powerups[i]
				break
			}
		}
		if selectedPowerup == nil {
			return
		}
		cost := int64(selectedPowerup.Price)
		if user.Score < cost {
			responseText := h.translator.Translate(user.LanguageCode, "market_not_enough_points", nil)
			h.sendMessage(query.Message.Chat.ID, responseText, tgbotapi.ModeHTML)
			return
		}
		h.storage.IncreaseUserScore(user.ID, -int(cost))
		h.storage.UpdateUserPowerUp(user.ID, selectedPowerup.ID, 1)
		var powerupName string
		if user.LanguageCode == "id" {
			powerupName = selectedPowerup.IDLocale.Name
		} else {
			powerupName = selectedPowerup.EN.Name
		}
		responseText := h.translator.Translate(user.LanguageCode, "powerup_purchase_success", map[string]string{"item": powerupName})
		h.sendMessage(query.Message.Chat.ID, responseText, tgbotapi.ModeHTML)

	case "main":
		deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)
		h.bot.Request(deleteMsg)
		h.handleMarketCommand(query.Message, user)
	}
}

func (h *BotHandler) handlePowerupActions(query *tgbotapi.CallbackQuery, user *storage.User, action string, args []string) {
	switch action {
	case "use":
		powerupID := args[0]
		h.usePowerup(query.Message, user, powerupID)
		deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)
		h.bot.Request(deleteMsg)
	}
}
// ▲▲▲ FUNGSI-FUNGSI BARU DITAMBAHKAN ▲▲▲

// ^^^ GANTI DENGAN FUNGSI INI ^^^


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

	if puzzle.RemainingSolution == "" {
		h.mu.Lock()
		delete(h.activePuzzles, message.Chat.ID)
		h.mu.Unlock()

		points := puzzle.Points
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

		playAgainButton := tgbotapi.NewInlineKeyboardButtonData(
			h.translator.Translate(user.LanguageCode, "play_again_button", nil),
			"play_again",
		)
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(playAgainButton))
		msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = markup
		h.bot.Send(msg)
	} else {
		params := map[string]string{"guessed_chars": result.CorrectlyGuessedChars}
		responseText := h.translator.Translate(user.LanguageCode, "partial_correct", params)
		h.sendMessage(message.Chat.ID, responseText, "")
	}
}
// ^^^ AKHIR PERUBAHAN ^^^


// ... Sisa file (ensureUserExists, handleCommand, dll) tetap sama
func (h *BotHandler) ensureUserExists(tgUser *tgbotapi.User) (*storage.User, error) {
	user, err := h.storage.GetUser(tgUser.ID)
	if err != nil {
		user = &storage.User{
			ID:           tgUser.ID,
			LanguageCode: h.config.DefaultLanguage,
		}
	}
	user.FirstName = tgUser.FirstName
	user.LastName = tgUser.LastName
	user.Username = tgUser.UserName
	if err := h.storage.UpsertUser(*user); err != nil {
		return nil, err
	}
	return user, nil
}

func (h *BotHandler) handleCommand(message *tgbotapi.Message, user *storage.User) {
	switch message.Command() {
	case "start":
		h.handleStartCommand(message, user)
	case "help":
		h.handleHelpCommand(message, user)
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
	case "market":
		h.handleMarketCommand(message, user)
	case "powerups":
		h.handlePowerupsCommand(message, user)
	case "surrender", "menyerah":
		h.handleSurrenderCommand(message, user)
	}
}

// vvv FUNGSI BARU DITAMBAHKAN vvv
// vvv AWAL PERUBAHAN vvv
func (h *BotHandler) handleMarketCommand(message *tgbotapi.Message, user *storage.User) {
	text := h.translator.Translate(user.LanguageCode, "market_intro", nil)
	themesButtonText := h.translator.Translate(user.LanguageCode, "market_category_themes", nil)
	powerupsButtonText := h.translator.Translate(user.LanguageCode, "market_category_powerups", nil)

	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(themesButtonText, "market_themes"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(powerupsButtonText, "market_powerups"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = markup
	h.bot.Send(msg)
}

// ▼▼▼ FUNGSI-FUNGSI BARU DITAMBAHKAN ▼▼▼
func (h *BotHandler) handlePowerupsCommand(message *tgbotapi.Message, user *storage.User) {
	args := strings.ToLower(strings.TrimSpace(message.CommandArguments()))
	if args == "reveal" {
		h.usePowerup(message, user, "reveal_letter")
		return
	}

	updatedUser, err := h.storage.GetUser(user.ID)
	if err != nil {
		log.Printf("Failed to get updated user for powerups: %v", err)
		return
	}

	if updatedUser.RevealLetter <= 0 {
		responseText := h.translator.Translate(user.LanguageCode, "no_powerups", nil)
		h.sendMessage(message.Chat.ID, responseText, "") 
		return
	}

	text := h.translator.Translate(user.LanguageCode, "powerups_intro", nil)
	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	if updatedUser.RevealLetter > 0 {
		var powerup *game.MarketItem
		for i := range h.powerupConfig.Powerups {
			if h.powerupConfig.Powerups[i].ID == "reveal_letter" {
				powerup = &h.powerupConfig.Powerups[i]
				break
			}
		}

		if powerup != nil {
			var powerupName string
			if user.LanguageCode == "id" {
				powerupName = powerup.IDLocale.Name
			} else {
				powerupName = powerup.EN.Name
			}
			params := map[string]string{
				"name":  powerupName,
				"count": strconv.Itoa(updatedUser.RevealLetter),
			}
			buttonText := h.translator.Translate(user.LanguageCode, "use_powerup_button", params)
			button := tgbotapi.NewInlineKeyboardButtonData(buttonText, "powerup_use_reveal_letter")
			keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(button))
		}
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeHTML // FIX: Menambahkan ParseMode HTML di sini.
	msg.ReplyMarkup = markup
	h.bot.Send(msg)
}

func (h *BotHandler) usePowerup(message *tgbotapi.Message, user *storage.User, powerupID string) {
	h.mu.Lock()
	puzzle, isActive := h.activePuzzles[message.Chat.ID]
	h.mu.Unlock()

	if !isActive {
		responseText := h.translator.Translate(user.LanguageCode, "powerup_no_active_puzzle", nil)
		h.sendMessage(message.Chat.ID, responseText, "")
		return
	}

	updatedUser, err := h.storage.GetUser(user.ID)
	if err != nil {
		log.Printf("Failed to get user to use powerup: %v", err)
		return
	}

	var hasPowerup bool
	if powerupID == "reveal_letter" && updatedUser.RevealLetter > 0 {
		hasPowerup = true
	}

	if !hasPowerup {
		responseText := h.translator.Translate(user.LanguageCode, "powerup_not_enough", nil)
		h.sendMessage(message.Chat.ID, responseText, tgbotapi.ModeHTML)
		return
	}

	revealedChar, success := puzzle.RevealRandomChar()
	if !success {
		responseText := h.translator.Translate(user.LanguageCode, "powerup_no_effect", nil)
		h.sendMessage(message.Chat.ID, responseText, "")
		return
	}

	h.storage.UpdateUserPowerUp(user.ID, powerupID, -1)

	params := map[string]string{"char": string(revealedChar)}
	responseText := h.translator.Translate(user.LanguageCode, "powerup_used_success", params)
	h.sendMessage(message.Chat.ID, responseText, tgbotapi.ModeHTML)

	newPuzzleText := "`" + puzzle.RenderDisplay() + "`"
	h.editMessage(message.Chat.ID, puzzle.MessageID, newPuzzleText, tgbotapi.ModeMarkdownV2)
}
// ▲▲▲ FUNGSI-FUNGSI BARU DITAMBAHKAN ▲▲▲

func (h *BotHandler) handleHelpCommand(message *tgbotapi.Message, user *storage.User) {
	text := h.translator.Translate(user.LanguageCode, "help_intro", nil)
	markup := h.buildHelpKeyboard(user.LanguageCode, "main")
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyMarkup = markup
	h.bot.Send(msg)
}

func (h *BotHandler) buildHelpKeyboard(langCode string, menuType string) tgbotapi.InlineKeyboardMarkup {
	if menuType == "back_only" {
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(h.translator.Translate(langCode, "help_button_back", nil), "help_main"),
			),
		)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(h.translator.Translate(langCode, "help_button_howtoplay", nil), "help_howtoplay"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(h.translator.Translate(langCode, "help_button_whatiscrypto", nil), "help_whatiscrypto"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(h.translator.Translate(langCode, "help_button_commands", nil), "help_commands"),
		),
	)
}

// vvv AWAL PERUBAHAN vvv
func (h *BotHandler) handleSurrenderCommand(message *tgbotapi.Message, user *storage.User) {
	h.mu.Lock()
	puzzle, isActive := h.activePuzzles[message.Chat.ID]
	if isActive {
		delete(h.activePuzzles, message.Chat.ID)
	}
	h.mu.Unlock()

	if !isActive {
		responseText := h.translator.Translate(user.LanguageCode, "no_active_puzzle", nil)
		h.sendMessage(message.Chat.ID, responseText, "")
		return
	}

	puzzle.RevealAll()
	finalText := "`" + puzzle.RenderDisplay() + "`"
	h.editMessage(message.Chat.ID, puzzle.MessageID, finalText, tgbotapi.ModeMarkdownV2)

	params := map[string]string{"answer": puzzle.Solution}
	responseText := h.translator.Translate(user.LanguageCode, "surrender_message", params)
	h.sendMessage(message.Chat.ID, responseText, tgbotapi.ModeHTML)
}
// ^^^ AKHIR PERUBAHAN ^^^

// vvv AWAL PERUBAHAN vvv
func (h *BotHandler) handleCryptoCommand(message *tgbotapi.Message, user *storage.User) {
	if !message.Chat.IsPrivate() {
		h.mu.Lock()
		_, ok := h.activePuzzles[message.Chat.ID]
		h.mu.Unlock()
		if ok {
			responseText := h.translator.Translate(user.LanguageCode, "puzzle_in_progress", nil)
			h.sendMessage(message.Chat.ID, responseText, "")
			return
		}
	}

	args := strings.ToLower(strings.TrimSpace(message.CommandArguments()))
	difficulty := "easy"
	validDifficulties := map[string]bool{"easy": true, "medium": true, "hard": true, "veryhard": true}

	if _, isValid := validDifficulties[args]; isValid && args != "" {
		difficulty = args
	}

	puzzle, err := h.gameSvc.GeneratePuzzle(difficulty)
	if err != nil {
		log.Printf("Failed to generate puzzle: %v", err)
		return
	}

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

	h.mu.Lock()
	h.activePuzzles[message.Chat.ID] = puzzle
	h.mu.Unlock()
}
// ^^^ AKHIR PERUBAHAN ^^^

func (h *BotHandler) editMessage(chatID int64, messageID int, text string, parseMode string) {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if parseMode != "" {
		msg.ParseMode = parseMode
	}
	h.bot.Request(msg)
}
func (h *BotHandler) handleStartCommand(message *tgbotapi.Message, user *storage.User) {
	params := map[string]string{"name": message.From.FirstName}
	responseText := h.translator.Translate(user.LanguageCode, "welcome", params)
	h.sendMessage(message.Chat.ID, responseText, tgbotapi.ModeHTML)
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
// vvv AWAL PERUBAHAN vvv
// vvv AWAL PERUBAHAN vvv
func (h *BotHandler) handleProfileCommand(message *tgbotapi.Message, user *storage.User) {
	updatedUser, err := h.storage.GetUser(user.ID)
	if err != nil {
		log.Printf("Failed to get updated user for profile: %v", err)
		updatedUser = user
	}

	params := map[string]string{
		"name":  updatedUser.FirstName,
		"score": strconv.FormatInt(updatedUser.Score, 10),
	}

	var selectedTheme *game.MarketItem
	for i := range h.themeConfig.Themes {
		if h.themeConfig.Themes[i].ID == updatedUser.ProfileTheme {
			selectedTheme = &h.themeConfig.Themes[i]
			break
		}
	}

	if selectedTheme == nil || selectedTheme.ID == "" {
		for i := range h.themeConfig.Themes {
			if h.themeConfig.Themes[i].ID == "default" {
				selectedTheme = &h.themeConfig.Themes[i]
				break
			}
		}
	}

	var localeData game.ItemLocale
	if updatedUser.LanguageCode == "id" {
		localeData = selectedTheme.IDLocale
	} else {
		localeData = selectedTheme.EN
	}

	responseText := localeData.Template
	for k, v := range params {
		responseText = strings.ReplaceAll(responseText, "{"+k+"}", v)
	}

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