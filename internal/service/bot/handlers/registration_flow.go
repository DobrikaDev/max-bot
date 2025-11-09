package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	userpb "DobrikaDev/max-bot/internal/generated/userpb"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	schemes "github.com/max-messenger/max-bot-api-client-go/schemes"
	"go.uber.org/zap"
)

func (h *MessageHandler) updateSessionMessage(ctx context.Context, session *registrationSession, text string, keyboard *maxbot.Keyboard) {
	if session.MessageID != "" {
		if err := h.editInteractiveMessage(ctx, session.ChatID, session.UserID, session.MessageID, text, keyboard); err == nil {
			h.sessions.upsert(session)
			return
		} else {
			h.logger.Warn("failed to edit registration message", zap.Error(err), zap.Int64("chat_id", session.ChatID), zap.Int64("user_id", session.UserID))
			return
		}
	}

	messageID, err := h.sendInteractiveMessage(ctx, session.ChatID, session.UserID, text, keyboard)
	if err != nil {
		h.logger.Error("failed to send registration message", zap.Error(err), zap.Int64("chat_id", session.ChatID), zap.String("text", text))
		return
	}

	session.MessageID = messageID
	h.sessions.upsert(session)
}

func emptyKeyboard() *maxbot.Keyboard {
	return nil
}

var agePayloadToAge = map[string]int32{
	callbackRegistrationAgeUnder18: 17,
	callbackRegistrationAge18_24:   21,
	callbackRegistrationAge25_34:   29,
	callbackRegistrationAge35_44:   39,
	callbackRegistrationAge45_54:   49,
	callbackRegistrationAge55_64:   59,
	callbackRegistrationAge65Plus:  70,
}

func (h *MessageHandler) ageKeyboard() *maxbot.Keyboard {
	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(h.messages.RegistrationAgeUnder18Button, schemes.DEFAULT, callbackRegistrationAgeUnder18).
		AddCallback(h.messages.RegistrationAge18_24Button, schemes.DEFAULT, callbackRegistrationAge18_24)
	keyboard.AddRow().
		AddCallback(h.messages.RegistrationAge25_34Button, schemes.DEFAULT, callbackRegistrationAge25_34).
		AddCallback(h.messages.RegistrationAge35_44Button, schemes.DEFAULT, callbackRegistrationAge35_44)
	keyboard.AddRow().
		AddCallback(h.messages.RegistrationAge45_54Button, schemes.DEFAULT, callbackRegistrationAge45_54).
		AddCallback(h.messages.RegistrationAge55_64Button, schemes.DEFAULT, callbackRegistrationAge55_64)
	keyboard.AddRow().
		AddCallback(h.messages.RegistrationAge65PlusButton, schemes.DEFAULT, callbackRegistrationAge65Plus)

	return keyboard
}

func (h *MessageHandler) startRegistration(ctx context.Context, userID, chatID int64, userName string, messageID string) {
	if session, ok := h.sessions.get(userID); ok && session.isInProgress() {
		if messageID != "" {
			session.MessageID = messageID
			session.ChatID = chatID
			h.sessions.upsert(session)
		}
		h.resumeRegistration(ctx, session)
		return
	}

	session := &registrationSession{
		UserID:    userID,
		ChatID:    chatID,
		UserName:  userName,
		MaxUserID: fmt.Sprintf("%d", userID),
		Current:   registrationStepAge,
		MessageID: messageID,
	}
	h.sessions.upsert(session)

	if messageID != "" {
		h.menus.delete(chatID)
	}

	h.promptForAge(ctx, session)
}

func (h *MessageHandler) resumeRegistration(ctx context.Context, session *registrationSession) {
	switch session.Current {
	case registrationStepAge:
		h.promptForAge(ctx, session)
	case registrationStepSex:
		h.promptForSex(ctx, session)
	case registrationStepLocation:
		h.promptForLocation(ctx, session)
	case registrationStepAbout:
		h.promptForAbout(ctx, session)
	default:
		if session.MessageID != "" {
			h.menus.set(session.ChatID, session.MessageID, session.UserID)
		}
		h.SendMainMenu(ctx, session.ChatID, session.UserID)
	}
}

func (h *MessageHandler) promptForAge(ctx context.Context, session *registrationSession) {
	h.updateSessionMessage(ctx, session, h.messages.RegistrationStartText, h.ageKeyboard())
}

func (h *MessageHandler) promptForSex(ctx context.Context, session *registrationSession) {
	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(h.messages.RegistrationSexMaleText, schemes.DEFAULT, callbackRegistrationSexMale).
		AddCallback(h.messages.RegistrationSexFemaleText, schemes.DEFAULT, callbackRegistrationSexFemale)

	h.updateSessionMessage(ctx, session, h.messages.RegistrationSexPrompt, keyboard)
}

func (h *MessageHandler) promptForLocation(ctx context.Context, session *registrationSession) {
	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddGeolocation(h.messages.RegistrationLocationGeoButton, true)
	keyboard.AddRow().
		AddCallback(h.messages.RegistrationLocationSkipButton, schemes.NEGATIVE, callbackRegistrationSkipLocation)

	h.updateSessionMessage(ctx, session, h.messages.RegistrationLocationPrompt, keyboard)
}

func (h *MessageHandler) promptForAbout(ctx context.Context, session *registrationSession) {
	h.updateSessionMessage(ctx, session, h.messages.RegistrationAboutPrompt, h.aboutKeyboard(session))
}

func (h *MessageHandler) tryHandleRegistrationMessage(ctx context.Context, update *schemes.MessageCreatedUpdate) bool {
	session, ok := h.sessions.get(update.Message.Sender.UserId)
	if !ok || !session.isInProgress() {
		return false
	}

	switch session.Current {
	case registrationStepAge:
		age, err := parseAge(update.GetText())
		if err != nil {
			h.updateSessionMessage(ctx, session, h.messages.RegistrationAgeRetryText, h.ageKeyboard())
			return true
		}

		session.Age = age
		session.Current = registrationStepSex
		h.sessions.upsert(session)
		h.promptForSex(ctx, session)

	case registrationStepLocation:
		if lat, lon, label, ok := extractLocation(update); ok {
			session.Latitude = lat
			session.Longitude = lon
			session.GeoLabel = label
			session.Current = registrationStepAbout
			session.Interests = make(map[int]bool)
			session.About = ""
			h.sessions.upsert(session)
			h.promptForAbout(ctx, session)
		} else {
			h.updateSessionMessage(ctx, session, h.messages.RegistrationLocationRetryText, emptyKeyboard())
		}

	case registrationStepAbout:
		text := strings.TrimSpace(update.GetText())
		if text != "" {
			session.About = text
			h.sessions.upsert(session)
			h.finalizeRegistration(ctx, session)
		} else {
			h.updateSessionMessage(ctx, session, h.messages.RegistrationAboutPrompt, h.aboutKeyboard(session))
		}

	default:
		h.logger.Debug("received message while in registration flow", zap.Int("step", int(session.Current)))
	}

	return true
}

func (h *MessageHandler) tryHandleRegistrationCallback(ctx context.Context, update *schemes.MessageCallbackUpdate) bool {
	payload := update.Callback.Payload
	if payload == "" {
		return false
	}

	if strings.HasPrefix(payload, callbackRegistrationAboutToggle+":") {
		h.handleAboutToggle(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	}

	switch payload {
	case callbackMainMenuRegistration:
		var chatID int64
		var messageID string
		if update.Message != nil {
			chatID = update.Message.Recipient.ChatId
			messageID = update.Message.Body.Mid
			if chatID != 0 {
				h.menus.delete(chatID)
			}
		} else if session, ok := h.sessions.get(update.Callback.User.UserId); ok {
			chatID = session.ChatID
		}
		if chatID == 0 {
			h.logger.Warn("registration callback without chat context")
			h.answerCallback(ctx, update.Callback.CallbackID)
			return true
		}
		h.startRegistration(ctx, update.Callback.User.UserId, chatID, update.Callback.User.Name, messageID)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackRegistrationSexMale, callbackRegistrationSexFemale:
		h.handleSexSelection(ctx, update, payload)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackRegistrationAboutConfirm:
		h.handleAboutConfirm(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackRegistrationSkipLocation:
		h.handleLocationSkip(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackRegistrationAgeUnder18,
		callbackRegistrationAge18_24,
		callbackRegistrationAge25_34,
		callbackRegistrationAge35_44,
		callbackRegistrationAge45_54,
		callbackRegistrationAge55_64,
		callbackRegistrationAge65Plus:
		h.handleAgeSelection(ctx, update, payload)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackRegistrationAboutToggle:
		h.handleAboutToggle(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	default:
		return false
	}
}

func (h *MessageHandler) handleSexSelection(ctx context.Context, update *schemes.MessageCallbackUpdate, payload string) {
	session, ok := h.sessions.get(update.Callback.User.UserId)
	if !ok {
		h.logger.Debug("sex selection without active session")
		return
	}

	if session.Current != registrationStepSex {
		h.logger.Debug("sex callback in unexpected step", zap.Int("step", int(session.Current)))
		return
	}

	if update.Message != nil && update.Message.Body.Mid != "" {
		session.MessageID = update.Message.Body.Mid
	}

	if payload == callbackRegistrationSexMale {
		session.Sex = userpb.Sex_SEX_MALE
	} else {
		session.Sex = userpb.Sex_SEX_FEMALE
	}

	session.Current = registrationStepLocation
	h.sessions.upsert(session)
	h.promptForLocation(ctx, session)
}

func (h *MessageHandler) handleAgeSelection(ctx context.Context, update *schemes.MessageCallbackUpdate, payload string) {
	session, ok := h.sessions.get(update.Callback.User.UserId)
	if !ok {
		h.logger.Debug("age selection without active session")
		return
	}

	if session.Current != registrationStepAge {
		h.logger.Debug("age callback in unexpected step", zap.Int("step", int(session.Current)))
		return
	}

	age, ok := agePayloadToAge[payload]
	if !ok {
		h.logger.Warn("unknown age payload", zap.String("payload", payload))
		return
	}

	if update.Message != nil && update.Message.Body.Mid != "" {
		session.MessageID = update.Message.Body.Mid
	}

	session.Age = age
	session.Current = registrationStepSex
	h.sessions.upsert(session)
	h.promptForSex(ctx, session)
}

func (h *MessageHandler) handleLocationSkip(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	session, ok := h.sessions.get(update.Callback.User.UserId)
	if !ok {
		return
	}

	if session.Current != registrationStepLocation {
		return
	}

	if update.Message != nil && update.Message.Body.Mid != "" {
		session.MessageID = update.Message.Body.Mid
	}

	session.GeoLabel = ""
	session.Current = registrationStepAbout
	session.Interests = make(map[int]bool)
	session.About = ""
	h.sessions.upsert(session)
	h.promptForAbout(ctx, session)
}

func (h *MessageHandler) handleAboutToggle(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	session, ok := h.sessions.get(update.Callback.User.UserId)
	if !ok {
		h.logger.Debug("about toggle without active session")
		return
	}
	if session.Current != registrationStepAbout {
		h.logger.Debug("about toggle in unexpected step", zap.Int("step", int(session.Current)))
		return
	}

	idxStr := update.Callback.Payload[len(callbackRegistrationAboutToggle)+1:]
	if idxStr == "" {
		h.logger.Warn("empty about option payload")
		return
	}

	index, err := strconv.Atoi(idxStr)
	if err != nil {
		h.logger.Warn("invalid about option index", zap.String("payload", idxStr), zap.Error(err))
		return
	}

	if index < 0 || index >= len(h.messages.RegistrationAboutOptions) {
		h.logger.Warn("about option index out of range", zap.Int("index", index))
		return
	}

	if update.Message != nil && update.Message.Body.Mid != "" {
		session.MessageID = update.Message.Body.Mid
	}

	if session.Interests == nil {
		session.Interests = make(map[int]bool)
	}
	if session.OriginalAboutOptionPrefix == nil {
		session.OriginalAboutOptionPrefix = make(map[int]string)
	}

	if session.Interests[index] {
		delete(session.Interests, index)
	} else {
		session.Interests[index] = true
	}

	h.sessions.upsert(session)
	h.updateSessionMessage(ctx, session, h.messages.RegistrationAboutPrompt, h.aboutKeyboard(session))
}

func (h *MessageHandler) handleAboutConfirm(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	session, ok := h.sessions.get(update.Callback.User.UserId)
	if !ok {
		return
	}
	if session.Current != registrationStepAbout {
		return
	}

	if update.Message != nil && update.Message.Body.Mid != "" {
		session.MessageID = update.Message.Body.Mid
	}

	if len(session.Interests) == 0 && session.About == "" {
		h.updateSessionMessage(ctx, session, h.messages.RegistrationAboutPrompt, h.aboutKeyboard(session))
		return
	}

	if len(session.Interests) > 0 {
		selected := make([]string, 0, len(session.Interests))
		for idx, option := range h.messages.RegistrationAboutOptions {
			if session.Interests[idx] {
				selected = append(selected, option)
			}
		}
		session.About = strings.Join(selected, "; ")
	}

	h.sessions.upsert(session)
	h.finalizeRegistration(ctx, session)
}

func (h *MessageHandler) finalizeRegistration(ctx context.Context, session *registrationSession) {
	if err := h.sendRegistrationToUserService(ctx, session); err != nil {
		h.logger.Error("failed to save registration", zap.Error(err), zap.Int64("user_id", session.UserID))
		h.updateSessionMessage(ctx, session, h.messages.RegistrationErrorText, emptyKeyboard())
		return
	}

	session.Current = registrationStepComplete

	summary := h.messages.RegistrationCompleteText
	if session.MessageID != "" {
		h.menus.set(session.ChatID, session.MessageID, session.UserID)
	}

	h.sessions.delete(session.UserID)
	h.SendMainMenu(ctx, session.ChatID, session.UserID, summary)
}

func extractLocation(update *schemes.MessageCreatedUpdate) (float64, float64, string, bool) {
	for _, attachment := range update.Message.Body.Attachments {
		switch v := attachment.(type) {
		case *schemes.LocationAttachment:
			return v.Latitude, v.Longitude, "", true
		case schemes.LocationAttachment:
			return v.Latitude, v.Longitude, "", true
		}
	}

	text := strings.TrimSpace(update.GetText())
	if text != "" {
		return 0, 0, text, true
	}

	return 0, 0, "", false
}

func (h *MessageHandler) answerCallback(ctx context.Context, callbackID string) {
	if callbackID == "" {
		return
	}

	if _, err := h.api.Messages.AnswerOnCallback(ctx, callbackID, &schemes.CallbackAnswer{}); err != nil && !isBenignAPIError(err) {
		h.logger.Warn("failed to answer callback", zap.Error(err), zap.String("callback_id", callbackID))
	}
}

func (h *MessageHandler) aboutKeyboard(session *registrationSession) *maxbot.Keyboard {
	keyboard := h.api.Messages.NewKeyboardBuilder()

	options := h.messages.RegistrationAboutOptions

	if session.OriginalAboutOptionPrefix == nil {
		session.OriginalAboutOptionPrefix = make(map[int]string, len(options))
		for i, option := range options {
			parts := strings.SplitN(option, " ", 2)
			if len(parts) == 2 {
				session.OriginalAboutOptionPrefix[i] = parts[0]
			} else {
				session.OriginalAboutOptionPrefix[i] = ""
			}
		}
	}

	labelText := func(idx int) string {
		option := options[idx]
		parts := strings.SplitN(option, " ", 2)
		if len(parts) != 2 {
			if session.Interests != nil && session.Interests[idx] {
				return "✅ " + option
			}
			return option
		}

		textWithoutEmoji := parts[1]
		if session.Interests != nil && session.Interests[idx] {
			return fmt.Sprintf("✅ %s", textWithoutEmoji)
		}

		if prefix, ok := session.OriginalAboutOptionPrefix[idx]; ok && prefix != "" {
			return fmt.Sprintf("%s %s", prefix, textWithoutEmoji)
		}

		return option
	}

	for i := 0; i < len(options); i += 2 {
		row := keyboard.AddRow()
		row.AddCallback(
			labelText(i),
			schemes.DEFAULT,
			fmt.Sprintf("%s:%d", callbackRegistrationAboutToggle, i),
		)
		if i+1 < len(options) {
			row.AddCallback(
				labelText(i+1),
				schemes.DEFAULT,
				fmt.Sprintf("%s:%d", callbackRegistrationAboutToggle, i+1),
			)
		}
	}

	keyboard.AddRow().
		AddCallback(h.messages.RegistrationAboutConfirmButton, schemes.POSITIVE, callbackRegistrationAboutConfirm)

	return keyboard
}
