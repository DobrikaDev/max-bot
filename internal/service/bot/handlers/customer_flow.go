package handlers

import (
	"context"
	"fmt"
	"strings"

	customerpb "DobrikaDev/max-bot/internal/generated/customerpb"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	schemes "github.com/max-messenger/max-bot-api-client-go/schemes"
	"go.uber.org/zap"
)

func (h *MessageHandler) tryHandleCustomerMessage(ctx context.Context, update *schemes.MessageCreatedUpdate) bool {
	session, ok := h.customerSessions.get(update.Message.Sender.UserId)
	if !ok || !session.isInProgress() {
		return false
	}

	text := strings.TrimSpace(update.GetText())

	switch session.Current {
	case customerStepName:
		if text == "" {
			retry := h.messages.CustomerNameRetryText
			if strings.TrimSpace(retry) == "" {
				retry = "Пожалуйста, укажи имя или название."
			}
			h.updateCustomerSessionMessage(ctx, session, retry, emptyKeyboard())
			return true
		}

		session.Name = text
		session.Current = customerStepAbout
		h.customerSessions.upsert(session)
		h.promptCustomerAbout(ctx, session)
	case customerStepAbout:
		if text == "" {
			retry := h.messages.CustomerAboutRetryText
			if strings.TrimSpace(retry) == "" {
				retry = "Пожалуйста, расскажи, какая помощь нужна."
			}
			h.updateCustomerSessionMessage(ctx, session, retry, emptyKeyboard())
			return true
		}

		session.About = text
		h.customerSessions.upsert(session)
		h.finalizeCustomerFlow(ctx, session)
	default:
		h.logger.Debug("customer flow received unexpected message", zap.Int("step", int(session.Current)))
	}

	return true
}

func (h *MessageHandler) tryHandleCustomerCallback(ctx context.Context, update *schemes.MessageCallbackUpdate) bool {
	payload := update.Callback.Payload
	if payload == "" {
		return false
	}

	switch payload {
	case callbackMainMenuNeedHelp:
		h.handleCustomerNeedHelp(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackCustomerTypeIndividual, callbackCustomerTypeBusiness:
		h.handleCustomerTypeSelection(ctx, update, payload)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackCustomerManageCreate:
		h.handleCustomerManageCreate(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackCustomerManageUpdate:
		h.handleCustomerManageUpdate(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackCustomerManageDelete:
		h.handleCustomerManageDelete(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackCustomerDeleteConfirm:
		h.handleCustomerDeleteConfirm(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackCustomerDeleteCancel:
		h.handleCustomerDeleteCancel(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	case callbackCustomerManageBack:
		h.handleCustomerManageBack(ctx, update)
		h.answerCallback(ctx, update.Callback.CallbackID)
		return true
	default:
		return false
	}
}

func (h *MessageHandler) handleCustomerNeedHelp(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	if update.Message == nil {
		h.logger.Warn("need help callback without message context")
		return
	}

	chatID := update.Message.Recipient.ChatId
	userID := update.Callback.User.UserId
	messageID := update.Message.Body.Mid

	if session, ok := h.customerSessions.get(userID); ok && session.isInProgress() {
		session.ChatID = chatID
		if messageID != "" {
			session.MessageID = messageID
		}
		h.customerSessions.upsert(session)
		h.resumeCustomerFlow(ctx, session)
		return
	}

	if h.customer == nil {
		text := h.messages.CustomerServiceUnavailableText
		if strings.TrimSpace(text) == "" {
			text = "Сервис заказчиков недоступен. Попробуй позже."
		}
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}

	customer, err := h.getCustomerByMaxID(ctx, fmt.Sprintf("%d", userID))
	if err != nil {
		text := h.messages.CustomerLookupErrorText
		if strings.TrimSpace(text) == "" {
			text = "Не удалось получить данные. Попробуй позже."
		}
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}

	if customer == nil {
		h.startCustomerFlow(ctx, userID, chatID, messageID, false, nil)
		return
	}

	h.showCustomerManageMenu(ctx, chatID, userID, customer, "")
}

func (h *MessageHandler) handleCustomerManageCreate(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	if update.Message == nil {
		return
	}

	userID := update.Callback.User.UserId
	chatID := update.Message.Recipient.ChatId

	if h.customer == nil {
		text := h.messages.CustomerServiceUnavailableText
		if strings.TrimSpace(text) == "" {
			text = "Сервис заказчиков недоступен. Попробуй позже."
		}
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}

	messageID := update.Message.Body.Mid

	h.startCustomerFlow(ctx, userID, chatID, messageID, false, nil)
}

func (h *MessageHandler) handleCustomerManageUpdate(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	if update.Message == nil {
		return
	}
	if h.customer == nil {
		h.logger.Warn("customer manage update requested but service client not configured")
		text := h.messages.CustomerServiceUnavailableText
		if strings.TrimSpace(text) == "" {
			text = "Сервис заказчиков недоступен. Попробуй позже."
		}
		h.renderMenu(ctx, update.Message.Recipient.ChatId, update.Callback.User.UserId, text, h.customerBackKeyboard())
		return
	}

	userID := update.Callback.User.UserId
	chatID := update.Message.Recipient.ChatId
	messageID := update.Message.Body.Mid

	customer, err := h.getCustomerByMaxID(ctx, fmt.Sprintf("%d", userID))
	if err != nil {
		text := h.messages.CustomerLookupErrorText
		if strings.TrimSpace(text) == "" {
			text = "Не удалось получить данные. Попробуй позже."
		}
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}

	if customer == nil {
		h.showCustomerEmptyMenu(ctx, chatID, userID, "")
		return
	}

	h.startCustomerFlow(ctx, userID, chatID, messageID, true, customer)
}

func (h *MessageHandler) handleCustomerManageDelete(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	if update.Message == nil {
		return
	}

	text := h.messages.CustomerDeleteConfirmText
	if strings.TrimSpace(text) == "" {
		text = "Удалить профиль заказчика? Это действие нельзя отменить."
	}

	confirm := h.messages.CustomerDeleteConfirmButton
	if strings.TrimSpace(confirm) == "" {
		confirm = "Удалить"
	}
	cancel := h.messages.CustomerDeleteCancelButton
	if strings.TrimSpace(cancel) == "" {
		cancel = "Отмена"
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(confirm, schemes.NEGATIVE, callbackCustomerDeleteConfirm)
	keyboard.AddRow().
		AddCallback(cancel, schemes.DEFAULT, callbackCustomerDeleteCancel)

	h.renderMenu(ctx, update.Message.Recipient.ChatId, update.Callback.User.UserId, text, keyboard)
}

func (h *MessageHandler) handleCustomerDeleteConfirm(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	if h.customer == nil || update.Message == nil {
		return
	}

	userID := update.Callback.User.UserId
	chatID := update.Message.Recipient.ChatId

	req := &customerpb.DeleteCustomerRequest{MaxId: fmt.Sprintf("%d", userID)}
	resp, err := h.customer.DeleteCustomer(ctx, req)
	if err != nil {
		h.logger.Error("failed to delete customer", zap.Error(err), zap.Int64("user_id", userID))
		text := h.messages.CustomerDeleteErrorText
		if strings.TrimSpace(text) == "" {
			text = "Не удалось удалить профиль. Попробуй позже."
		}
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}

	if e := resp.GetError(); e != nil {
		h.logger.Warn("customer service returned error on delete", zap.String("message", e.GetMessage()))
		text := h.messages.CustomerDeleteErrorText
		if strings.TrimSpace(text) == "" {
			text = "Не удалось удалить профиль. Попробуй позже."
		}
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}

	h.customerSessions.delete(userID)

	success := h.messages.CustomerDeleteSuccessText
	if strings.TrimSpace(success) == "" {
		success = "Профиль заказчика удалён."
	}
	h.showCustomerEmptyMenu(ctx, chatID, userID, success)
}

func (h *MessageHandler) handleCustomerDeleteCancel(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	if update.Message == nil {
		return
	}

	userID := update.Callback.User.UserId
	chatID := update.Message.Recipient.ChatId

	customer, err := h.getCustomerByMaxID(ctx, fmt.Sprintf("%d", userID))
	if err != nil || customer == nil {
		h.showCustomerEmptyMenu(ctx, chatID, userID, "")
		return
	}

	h.showCustomerManageMenu(ctx, chatID, userID, customer, "")
}

func (h *MessageHandler) handleCustomerManageBack(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Recipient.ChatId
	userID := update.Callback.User.UserId
	if update.Message.Body.Mid != "" {
		h.menus.set(chatID, update.Message.Body.Mid, userID)
	}
	h.SendMainMenu(ctx, chatID, userID)
}

func (h *MessageHandler) startCustomerFlow(ctx context.Context, userID, chatID int64, messageID string, existing bool, existingCustomer *customerpb.Customer) {
	session := &customerSession{
		UserID:    userID,
		ChatID:    chatID,
		MessageID: messageID,
		MaxUserID: fmt.Sprintf("%d", userID),
		Existing:  existing,
		Current:   customerStepType,
	}

	if existing && existingCustomer != nil {
		session.Type = existingCustomer.GetType()
		session.Name = strings.TrimSpace(existingCustomer.GetName())
		session.About = strings.TrimSpace(existingCustomer.GetAbout())
	}

	h.customerSessions.upsert(session)
	if messageID != "" {
		h.menus.delete(chatID)
	}

	h.promptCustomerType(ctx, session)
}

func (h *MessageHandler) resumeCustomerFlow(ctx context.Context, session *customerSession) {
	switch session.Current {
	case customerStepType:
		h.promptCustomerType(ctx, session)
	case customerStepName:
		h.promptCustomerName(ctx, session)
	case customerStepAbout:
		h.promptCustomerAbout(ctx, session)
	default:
		customer, err := h.getCustomerByMaxID(ctx, session.MaxUserID)
		if err != nil || customer == nil {
			h.showCustomerEmptyMenu(ctx, session.ChatID, session.UserID, "")
			return
		}
		h.showCustomerManageMenu(ctx, session.ChatID, session.UserID, customer, "")
	}
}

func (h *MessageHandler) promptCustomerType(ctx context.Context, session *customerSession) {
	prompt := h.messages.CustomerTypePrompt
	if strings.TrimSpace(prompt) == "" {
		prompt = "Кто будет получать помощь?"
	}

	if !session.Existing {
		intro := strings.TrimSpace(h.messages.CustomerFormIntroText)
		if intro != "" {
			prompt = fmt.Sprintf("%s\n\n%s", intro, prompt)
		}
	} else if session.Type != customerpb.CustomerType_CUSTOMER_TYPE_UNSPECIFIED {
		current := strings.TrimSpace(h.customerTypeLabel(session.Type))
		if current != "" {
			prompt = fmt.Sprintf("%s\n\n*Сейчас указано:* %s", prompt, current)
		}
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(h.messages.CustomerTypeIndividualButton, schemes.DEFAULT, callbackCustomerTypeIndividual)
	keyboard.AddRow().
		AddCallback(h.messages.CustomerTypeBusinessButton, schemes.DEFAULT, callbackCustomerTypeBusiness)

	h.updateCustomerSessionMessage(ctx, session, prompt, keyboard)
}

func (h *MessageHandler) handleCustomerTypeSelection(ctx context.Context, update *schemes.MessageCallbackUpdate, payload string) {
	session, ok := h.customerSessions.get(update.Callback.User.UserId)
	if !ok || !session.isInProgress() {
		h.logger.Debug("customer type selection without active session")
		return
	}

	if update.Message != nil {
		session.ChatID = update.Message.Recipient.ChatId
		if update.Message.Body.Mid != "" {
			session.MessageID = update.Message.Body.Mid
		}
	}

	switch payload {
	case callbackCustomerTypeIndividual:
		session.Type = customerpb.CustomerType_CUSTOMER_TYPE_INDIVIDUAL
	case callbackCustomerTypeBusiness:
		session.Type = customerpb.CustomerType_CUSTOMER_TYPE_BUSINESS
	default:
		h.logger.Warn("unknown customer type payload", zap.String("payload", payload))
		return
	}

	session.Current = customerStepName
	h.customerSessions.upsert(session)
	h.promptCustomerName(ctx, session)
}

func (h *MessageHandler) promptCustomerName(ctx context.Context, session *customerSession) {
	text := h.messages.CustomerNamePrompt
	if strings.TrimSpace(text) == "" {
		text = "Как тебя зовут? Если профиль для организации — укажи название."
	}

	if session.Existing && session.Name != "" {
		text = fmt.Sprintf("%s\n\n*Сейчас:* %s", text, session.Name)
	}

	h.updateCustomerSessionMessage(ctx, session, text, emptyKeyboard())
}

func (h *MessageHandler) promptCustomerAbout(ctx context.Context, session *customerSession) {
	text := h.messages.CustomerAboutPrompt
	if strings.TrimSpace(text) == "" {
		text = "Опиши, какая помощь нужна."
	}

	if session.Existing && session.About != "" {
		text = fmt.Sprintf("%s\n\n*Сейчас:* %s", text, session.About)
	}

	h.updateCustomerSessionMessage(ctx, session, text, emptyKeyboard())
}

func (h *MessageHandler) finalizeCustomerFlow(ctx context.Context, session *customerSession) {
	if h.customer == nil {
		errorText := h.messages.CustomerServiceUnavailableText
		if strings.TrimSpace(errorText) == "" {
			errorText = "Сервис заказчиков недоступен. Попробуй позже."
		}
		h.updateCustomerSessionMessage(ctx, session, errorText, emptyKeyboard())
		h.customerSessions.delete(session.UserID)
		return
	}

	if session.Type == customerpb.CustomerType_CUSTOMER_TYPE_UNSPECIFIED ||
		strings.TrimSpace(session.Name) == "" ||
		strings.TrimSpace(session.About) == "" {
		h.logger.Warn("customer session incomplete on finalize", zap.Int64("user_id", session.UserID))
		h.promptCustomerType(ctx, session)
		return
	}

	if err := h.saveCustomer(ctx, session); err != nil {
		h.logger.Error("failed to save customer", zap.Error(err), zap.Int64("user_id", session.UserID))
		errorText := h.messages.CustomerSaveErrorText
		if strings.TrimSpace(errorText) == "" {
			errorText = "Не получилось сохранить профиль. Попробуй позже."
		}
		h.updateCustomerSessionMessage(ctx, session, errorText, emptyKeyboard())
		return
	}

	session.Current = customerStepComplete
	h.customerSessions.delete(session.UserID)

	success := h.messages.CustomerCreateSuccessText
	if session.Existing {
		success = h.messages.CustomerUpdateSuccessText
	}
	if strings.TrimSpace(success) == "" {
		if session.Existing {
			success = "Профиль заказчика обновлён."
		} else {
			success = "Профиль заказчика сохранён."
		}
	}

	customer := &customerpb.Customer{
		MaxId: session.MaxUserID,
		Name:  session.Name,
		About: session.About,
		Type:  session.Type,
	}

	h.showCustomerManageMenu(ctx, session.ChatID, session.UserID, customer, success)
}

func (h *MessageHandler) saveCustomer(ctx context.Context, session *customerSession) error {
	customer := &customerpb.Customer{
		MaxId: session.MaxUserID,
		Name:  session.Name,
		About: session.About,
		Type:  session.Type,
	}

	if session.Existing {
		resp, err := h.customer.UpdateCustomer(ctx, &customerpb.UpdateCustomerRequest{Customer: customer})
		if err != nil {
			return err
		}
		if e := resp.GetError(); e != nil {
			return fmt.Errorf("customer service error: %s", e.GetMessage())
		}
		return nil
	}

	resp, err := h.customer.CreateCustomer(ctx, &customerpb.CreateCustomerRequest{Customer: customer})
	if err != nil {
		return err
	}
	if e := resp.GetError(); e != nil {
		return fmt.Errorf("customer service error: %s", e.GetMessage())
	}

	return nil
}

func (h *MessageHandler) showCustomerManageMenu(ctx context.Context, chatID, userID int64, customer *customerpb.Customer, intro string) {
	if customer == nil {
		h.showCustomerEmptyMenu(ctx, chatID, userID, intro)
		return
	}

	title := h.messages.CustomerSummaryTitle
	if strings.TrimSpace(title) == "" {
		title = "Профиль заказчика:"
	}

	summaryTemplate := h.messages.CustomerSummaryTemplate
	if strings.TrimSpace(summaryTemplate) == "" {
		summaryTemplate = "*Тип:* %s\n*Имя или название:* %s\n*Описание запроса:* %s"
	}

	about := strings.TrimSpace(customer.GetAbout())
	if about == "" {
		about = "—"
	}
	name := strings.TrimSpace(customer.GetName())
	if name == "" {
		name = "—"
	}

	summary := fmt.Sprintf(summaryTemplate, h.customerTypeLabel(customer.GetType()), name, about)

	builder := strings.Builder{}
	if strings.TrimSpace(intro) != "" {
		builder.WriteString(strings.TrimSpace(intro))
		builder.WriteString("\n\n")
	}
	builder.WriteString(strings.TrimSpace(title))
	builder.WriteString("\n")
	builder.WriteString(summary)

	updateLabel := h.messages.CustomerManageUpdateButton
	if strings.TrimSpace(updateLabel) == "" {
		updateLabel = "Обновить профиль"
	}
	deleteLabel := h.messages.CustomerManageDeleteButton
	if strings.TrimSpace(deleteLabel) == "" {
		deleteLabel = "Удалить профиль"
	}
	backLabel := h.messages.CustomerManageBackButton
	if strings.TrimSpace(backLabel) == "" {
		backLabel = "⬅️ Назад в меню"
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(updateLabel, schemes.DEFAULT, callbackCustomerManageUpdate)
	keyboard.AddRow().
		AddCallback(deleteLabel, schemes.NEGATIVE, callbackCustomerManageDelete)
	keyboard.AddRow().
		AddCallback(backLabel, schemes.DEFAULT, callbackCustomerManageBack)

	h.renderMenu(ctx, chatID, userID, builder.String(), keyboard)
}

func (h *MessageHandler) showCustomerEmptyMenu(ctx context.Context, chatID, userID int64, intro string) {
	text := h.messages.CustomerFormIntroText
	if strings.TrimSpace(intro) != "" {
		text = intro
	}
	if strings.TrimSpace(text) == "" {
		text = "Расскажи о заказчике, чтобы мы сформировали профиль и смогли быстрее помочь."
	}

	createLabel := h.messages.CustomerManageCreateButton
	if strings.TrimSpace(createLabel) == "" {
		createLabel = "Заполнить профиль"
	}
	backLabel := h.messages.CustomerManageBackButton
	if strings.TrimSpace(backLabel) == "" {
		backLabel = "⬅️ Назад в меню"
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(createLabel, schemes.POSITIVE, callbackCustomerManageCreate)
	keyboard.AddRow().
		AddCallback(backLabel, schemes.DEFAULT, callbackCustomerManageBack)

	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) customerBackKeyboard() *maxbot.Keyboard {
	label := h.messages.CustomerManageBackButton
	if strings.TrimSpace(label) == "" {
		label = "⬅️ Назад в меню"
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(label, schemes.DEFAULT, callbackCustomerManageBack)
	return keyboard
}

func (h *MessageHandler) customerTypeLabel(customerType customerpb.CustomerType) string {
	switch customerType {
	case customerpb.CustomerType_CUSTOMER_TYPE_BUSINESS:
		if label := strings.TrimSpace(h.messages.CustomerTypeBusinessLabel); label != "" {
			return label
		}
		return "Организация"
	case customerpb.CustomerType_CUSTOMER_TYPE_INDIVIDUAL:
		if label := strings.TrimSpace(h.messages.CustomerTypeIndividualLabel); label != "" {
			return label
		}
		return "Частное лицо"
	default:
		return ""
	}
}

func (h *MessageHandler) updateCustomerSessionMessage(ctx context.Context, session *customerSession, text string, keyboard *maxbot.Keyboard) {
	if session.MessageID != "" {
		if err := h.editInteractiveMessage(ctx, session.ChatID, session.UserID, session.MessageID, text, keyboard); err == nil {
			h.customerSessions.upsert(session)
			return
		} else {
			h.logger.Warn("failed to edit customer message", zap.Error(err), zap.Int64("chat_id", session.ChatID))
		}
	}

	messageID, err := h.sendInteractiveMessage(ctx, session.ChatID, session.UserID, text, keyboard)
	if err != nil {
		h.logger.Error("failed to send customer message", zap.Error(err), zap.Int64("chat_id", session.ChatID))
		return
	}

	session.MessageID = messageID
	h.customerSessions.upsert(session)
}

func (h *MessageHandler) getCustomerByMaxID(ctx context.Context, maxID string) (*customerpb.Customer, error) {
	if h.customer == nil {
		return nil, fmt.Errorf("customer service client is not configured")
	}

	resp, err := h.customer.GetCustomerByMaxID(ctx, &customerpb.GetCustomerByMaxIDRequest{MaxId: maxID})
	if err != nil {
		return nil, err
	}

	if e := resp.GetError(); e != nil {
		if e.GetCode() == customerpb.ErrorCode_ERROR_CODE_NOT_FOUND {
			return nil, nil
		}
		return nil, fmt.Errorf("customer service error: %s", e.GetMessage())
	}

	return resp.GetCustomer(), nil
}
