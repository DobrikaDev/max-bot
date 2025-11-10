package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	customerpb "DobrikaDev/max-bot/internal/generated/customerpb"
	taskpb "DobrikaDev/max-bot/internal/generated/taskpb"
	userpb "DobrikaDev/max-bot/internal/generated/userpb"
	"DobrikaDev/max-bot/internal/locales"
	"DobrikaDev/max-bot/utils/config"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	schemes "github.com/max-messenger/max-bot-api-client-go/schemes"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MessageHandler struct {
	api      *maxbot.Api
	cfg      *config.Config
	logger   *zap.Logger
	user     userpb.UserServiceClient
	customer customerpb.CustomerServiceClient
	task     taskpb.TaskServiceClient
	messages locales.Messages

	sessions         *sessionStore
	customerSessions *customerSessionStore
	taskSessions     *taskSessionStore
	menus            *menuStore

	httpClient *http.Client
	apiBaseURL string
	apiVersion string
}

const messageFormatMarkdown = "markdown"

type messageEditPayload struct {
	Text        string        `json:"text,omitempty"`
	Format      string        `json:"format,omitempty"`
	Attachments []interface{} `json:"attachments"`
}

func NewMessageHandler(api *maxbot.Api, cfg *config.Config, logger *zap.Logger) *MessageHandler {
	handler := &MessageHandler{
		api:              api,
		cfg:              cfg,
		logger:           logger,
		sessions:         newSessionStore(),
		customerSessions: newCustomerSessionStore(),
		taskSessions:     newTaskSessionStore(),
		menus:            newMenuStore(),
		httpClient:       &http.Client{Timeout: 10 * time.Second},
		apiBaseURL:       "https://botapi.max.ru",
		apiVersion:       "1.2.5",
	}

	msgs, err := locales.Load()
	if err != nil {
		logger.Warn("failed to load locales", zap.Error(err))
	}
	handler.messages = msgs

	if cfg.MaxToken == "" {
		logger.Warn("MAX token is empty; message editing will fail")
	}

	if cfg.UserServiceURL == "" {
		logger.Warn("user service URL is not configured; registration completion will be skipped")
	} else if conn, err := grpc.Dial(cfg.UserServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		logger.Error("failed to connect to user service", zap.Error(err))
	} else {
		handler.user = userpb.NewUserServiceClient(conn)
	}

	if cfg.CustomerServiceURL == "" {
		logger.Warn("customer service URL is not configured; need help flow will be disabled")
	} else if conn, err := grpc.Dial(cfg.CustomerServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		logger.Error("failed to connect to customer service", zap.Error(err))
	} else {
		handler.customer = customerpb.NewCustomerServiceClient(conn)
	}

	if cfg.TaskServiceURL == "" {
		logger.Warn("task service URL is not configured; task features will be disabled")
	} else if conn, err := grpc.Dial(cfg.TaskServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		logger.Error("failed to connect to task service", zap.Error(err))
	} else {
		handler.task = taskpb.NewTaskServiceClient(conn)
	}

	return handler
}

func (h *MessageHandler) HandleMessage(ctx context.Context, message *schemes.MessageCreatedUpdate) {
	h.logger.Info("Received message", zap.Any("message", message))

	if !h.ensureUserContext(ctx, message) {
		return
	}

	if h.tryHandleTaskCreationMessage(ctx, message) {
		return
	}

	if h.tryHandleCustomerMessage(ctx, message) {
		return
	}

	if h.tryHandleRegistrationMessage(ctx, message) {
		return
	}

	if h.isRegistrationTrigger(message) {
		h.startRegistration(ctx, message.Message.Sender.UserId, message.Message.Recipient.ChatId, message.Message.Sender.Name, "")
		return
	}

	if h.isStartCommand(message) {
		h.menus.delete(message.Message.Recipient.ChatId)
		h.SendMainMenu(ctx, message.Message.Recipient.ChatId, message.Message.Sender.UserId)
		return
	}
}
func (h *MessageHandler) HandleCallbackQuery(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate) {
	h.logger.Info("Received callback query", zap.Any("callbackQuery", callbackQuery))
	if h.tryHandleRegistrationCallback(ctx, callbackQuery) {
		return
	}

	if h.tryHandleCustomerCallback(ctx, callbackQuery) {
		return
	}

	if h.handleMainMenuCallback(ctx, callbackQuery) {
		return
	}

	if callbackQuery.Message != nil {
		chatID := callbackQuery.Message.Recipient.ChatId
		userID := callbackQuery.Message.Recipient.UserId
		if callbackQuery.Message.Body.Mid != "" {
			h.menus.set(chatID, callbackQuery.Message.Body.Mid, userID)
		}
		h.SendMainMenu(ctx, chatID, userID)
	}
}

func (h *MessageHandler) SendMainMenu(ctx context.Context, chatID, userID int64, intro ...string) {
	text := h.messages.MainMenuText
	if len(intro) > 0 && strings.TrimSpace(intro[0]) != "" {
		text = intro[0] + "\n\n" + text
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	if len(h.messages.MainMenuButtons) >= 2 {
		keyboard.AddRow().
			AddCallback(h.messages.MainMenuButtons[0], schemes.POSITIVE, callbackMainMenuHelp)
		keyboard.AddRow().
			AddCallback(h.messages.MainMenuButtons[1], schemes.DEFAULT, callbackMainMenuNeedHelp)
	}
	if len(h.messages.MainMenuButtons) >= 3 {
		keyboard.AddRow().
			AddCallback(h.messages.MainMenuButtons[2], schemes.DEFAULT, callbackMainMenuProfile)
	}
	if len(h.messages.MainMenuButtons) >= 4 {
		keyboard.AddRow().
			AddCallback(h.messages.MainMenuButtons[3], schemes.DEFAULT, callbackMainMenuAbout)
	}

	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) isStartCommand(message *schemes.MessageCreatedUpdate) bool {
	text := strings.TrimSpace(strings.ToLower(message.GetText()))
	return text == "/start" || text == "start" || text == "меню"
}

func (h *MessageHandler) isRegistrationTrigger(message *schemes.MessageCreatedUpdate) bool {
	text := strings.TrimSpace(strings.ToLower(message.GetText()))
	if text == "регистрация" || text == "хочу помогать" {
		return true
	}

	command := message.GetCommand()
	return strings.HasPrefix(command, "/register")
}

func parseAge(text string) (int32, error) {
	age, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, fmt.Errorf("не получилось распознать возраст: %w", err)
	}

	if age < 12 || age > 110 {
		return 0, fmt.Errorf("возраст вне допустимого диапазона")
	}

	return int32(age), nil
}

func isBenignAPIError(err error) bool {
	if err == nil {
		return true
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return true
	}
	if strings.HasPrefix(msg, "HTTP 400") {
		return true
	}
	return false
}

func (h *MessageHandler) ensureUserContext(ctx context.Context, message *schemes.MessageCreatedUpdate) bool {
	if h.user == nil {
		return true
	}

	if session, ok := h.sessions.get(message.Message.Sender.UserId); ok && session.isInProgress() {
		return true
	}

	if h.isRegistrationTrigger(message) {
		return true
	}

	exists, err := h.userExists(ctx, fmt.Sprintf("%d", message.Message.Sender.UserId))
	if err != nil {
		h.logger.Warn("failed to check user profile", zap.Error(err))
		return true
	}

	if !exists {
		h.SendJoinMenu(ctx, message.Message.Recipient.ChatId, message.Message.Sender.UserId)
		return false
	}

	return true
}

func (h *MessageHandler) userExists(ctx context.Context, maxID string) (bool, error) {
	if h.user == nil {
		return false, fmt.Errorf("user service client is not configured")
	}

	resp, err := h.user.GetUserByMaxID(ctx, &userpb.GetUserByMaxIDRequest{MaxId: maxID})
	if err != nil {
		return false, err
	}

	if resp.GetError() != nil {
		if resp.GetError().GetCode() == userpb.ErrorCode_ERROR_CODE_NOT_FOUND {
			return false, nil
		}

		return false, fmt.Errorf("user service error: %s", resp.GetError().GetMessage())
	}

	return true, nil
}

func (h *MessageHandler) SendJoinMenu(ctx context.Context, chatID, userID int64) {
	text := h.messages.NewUserWelcomeText

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(h.messages.NewUserJoinButton, schemes.POSITIVE, callbackMainMenuRegistration)

	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) renderMenu(ctx context.Context, chatID, userID int64, text string, keyboard *maxbot.Keyboard) {
	if entry, ok := h.menus.get(chatID); ok && entry.MessageID != "" {
		if err := h.editInteractiveMessage(ctx, chatID, entry.UserID, entry.MessageID, text, keyboard); err == nil {
			h.menus.set(chatID, entry.MessageID, userID)
			return
		} else {
			h.logger.Warn("failed to update menu message", zap.Error(err), zap.Int64("chat_id", chatID))
		}
		h.menus.delete(chatID)
	}

	messageID, err := h.sendInteractiveMessage(ctx, chatID, userID, text, keyboard)
	if err != nil {
		h.logger.Error("failed to send menu message", zap.Error(err), zap.Int64("chat_id", chatID))
		return
	}

	h.menus.set(chatID, messageID, userID)
}

func (h *MessageHandler) sendInteractiveMessage(ctx context.Context, chatID, userID int64, text string, keyboard *maxbot.Keyboard) (string, error) {
	msg := maxbot.NewMessage().
		SetChat(chatID).
		SetText(text).
		SetFormat(messageFormatMarkdown)

	if userID != 0 {
		msg.SetUser(userID)
	}

	if keyboard != nil {
		msg.AddKeyboard(keyboard)
	}

	sent, err := h.api.Messages.SendMessageResult(ctx, msg)
	if err != nil {
		return "", err
	}

	return sent.Body.Mid, nil
}

func (h *MessageHandler) editInteractiveMessage(ctx context.Context, chatID, userID int64, messageID, text string, keyboard *maxbot.Keyboard) error {
	if messageID == "" {
		return fmt.Errorf("message id is empty")
	}

	body := h.buildMessageBody(text, keyboard)
	return h.editMessageRaw(ctx, messageID, body)
}

func (h *MessageHandler) buildMessageBody(text string, keyboard *maxbot.Keyboard) *messageEditPayload {
	payload := &messageEditPayload{
		Text:   text,
		Format: messageFormatMarkdown,
	}

	if keyboard != nil {
		payload.Attachments = []interface{}{schemes.NewInlineKeyboardAttachmentRequest(keyboard.Build())}
	} else {
		payload.Attachments = []interface{}{}
	}

	return payload
}

func (h *MessageHandler) editMessageRaw(ctx context.Context, messageID string, body *messageEditPayload) error {
	if h.cfg.MaxToken == "" {
		return fmt.Errorf("max token is empty")
	}

	if body == nil {
		body = &messageEditPayload{
			Attachments: []interface{}{},
		}
	}

	query := url.Values{}
	query.Set("message_id", messageID)
	query.Set("access_token", h.cfg.MaxToken)
	query.Set("v", h.apiVersion)

	u := h.apiBaseURL
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	u += "messages"

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal message body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s?%s", u, query.Encode()), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create edit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "max-bot-dynamic-menu/1.0")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute edit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var result schemes.SimpleQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode edit response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("edit response unsuccessful: %s", result.Message)
	}

	return nil
}

func (h *MessageHandler) handleMainMenuCallback(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate) bool {
	payload := callbackQuery.Callback.Payload
	if payload == "" || callbackQuery.Message == nil {
		return false
	}

	chatID := callbackQuery.Message.Recipient.ChatId
	userID := callbackQuery.Callback.User.UserId

	switch {
	case strings.HasPrefix(payload, callbackVolunteerTaskView+":"):
		h.handleVolunteerTaskView(ctx, callbackQuery, strings.TrimPrefix(payload, callbackVolunteerTaskView+":"))
		return true
	case strings.HasPrefix(payload, callbackVolunteerTaskJoin+":"):
		h.handleVolunteerTaskJoin(ctx, callbackQuery, strings.TrimPrefix(payload, callbackVolunteerTaskJoin+":"))
		return true
	case strings.HasPrefix(payload, callbackVolunteerTaskLeave+":"):
		h.handleVolunteerTaskLeave(ctx, callbackQuery, strings.TrimPrefix(payload, callbackVolunteerTaskLeave+":"))
		return true
	case strings.HasPrefix(payload, callbackVolunteerTaskConfirm+":"):
		h.handleVolunteerTaskConfirm(ctx, callbackQuery, strings.TrimPrefix(payload, callbackVolunteerTaskConfirm+":"))
		return true
	case strings.HasPrefix(payload, callbackCustomerTaskView+":"):
		h.handleCustomerTaskView(ctx, callbackQuery, strings.TrimPrefix(payload, callbackCustomerTaskView+":"))
		return true
	case strings.HasPrefix(payload, callbackCustomerTaskApprove+":"):
		h.handleCustomerTaskApprove(ctx, callbackQuery, strings.TrimPrefix(payload, callbackCustomerTaskApprove+":"))
		return true
	case strings.HasPrefix(payload, callbackCustomerTaskReject+":"):
		h.handleCustomerTaskReject(ctx, callbackQuery, strings.TrimPrefix(payload, callbackCustomerTaskReject+":"))
		return true
	}

	switch payload {
	case callbackMainMenuProfile:
		h.showProfile(ctx, chatID, userID)
	case callbackMainMenuHelp:
		h.showVolunteerMenu(ctx, chatID, userID)
	case callbackMainMenuAbout:
		h.showAboutDobrikaMenu(ctx, chatID, userID)
	case callbackVolunteerOnDemand:
		h.showVolunteerTasksList(ctx, chatID, userID, h.messages.VolunteerOnDemandPlaceholder)
	case callbackVolunteerTasks:
		h.showVolunteerTasksList(ctx, chatID, userID, h.messages.VolunteerTasksPlaceholder)
	case callbackVolunteerBack:
		h.showVolunteerMenu(ctx, chatID, userID)
	case callbackProfileCoins:
		h.showProfileCoinsMenu(ctx, chatID, userID)
	case callbackProfileHistory:
		h.showProfileHistory(ctx, chatID, userID)
	case callbackProfileEdit:
		h.showProfileEdit(ctx, chatID, userID)
	case callbackProfileSecurity:
		h.showProfileSecurity(ctx, chatID, userID)
	case callbackProfileBack:
		h.SendMainMenu(ctx, chatID, userID)
	case callbackCoinsHowToGet:
		h.showCoinsHowToGet(ctx, chatID, userID)
	case callbackCoinsHowToSpend:
		h.showCoinsHowToSpend(ctx, chatID, userID)
	case callbackCoinsLevels:
		h.showCoinsLevels(ctx, chatID, userID)
	case callbackAboutHowItWorks:
		h.renderMenu(ctx, chatID, userID, h.messages.AboutDobrikaHowText, h.aboutMenuKeyboard())
	case callbackAboutRules:
		h.renderMenu(ctx, chatID, userID, h.messages.AboutDobrikaRulesText, h.aboutMenuKeyboard())
	case callbackAboutInitiator:
		h.renderMenu(ctx, chatID, userID, h.messages.AboutDobrikaInitiatorText, h.aboutMenuKeyboard())
	case callbackAboutSupport:
		h.renderMenu(ctx, chatID, userID, h.messages.AboutDobrikaSupportText, h.aboutMenuKeyboard())
	case callbackAboutBack:
		h.SendMainMenu(ctx, chatID, userID)
	default:
		return false
	}

	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)
	return true
}

func (h *MessageHandler) showProfile(ctx context.Context, chatID, userID int64) {
	text, err := h.buildProfileText(ctx, userID)
	if err != nil {
		h.logger.Error("failed to build profile text", zap.Error(err), zap.Int64("user_id", userID))
		text = "Не удалось получить данные профиля. Попробуйте позже."
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(h.messages.ProfileHistoryButton, schemes.DEFAULT, callbackProfileHistory).
		AddCallback(h.messages.ProfileEditButton, schemes.DEFAULT, callbackProfileEdit)
	keyboard.AddRow().
		AddCallback(h.messages.ProfileCoinsButton, schemes.DEFAULT, callbackProfileCoins).
		AddCallback(h.messages.ProfileSecurityButton, schemes.DEFAULT, callbackProfileSecurity)
	keyboard.AddRow().
		AddCallback(h.messages.ProfileBackButton, schemes.DEFAULT, callbackProfileBack)

	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) showVolunteerMenu(ctx context.Context, chatID, userID int64) {
	text := h.messages.VolunteerMenuIntro
	if strings.TrimSpace(text) == "" {
		text = "💚 Выбери, как хочешь помочь:"
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(h.messages.VolunteerMenuOnDemandButton, schemes.DEFAULT, callbackVolunteerOnDemand)
	keyboard.AddRow().
		AddCallback(h.messages.VolunteerMenuTasksButton, schemes.DEFAULT, callbackVolunteerTasks)
	keyboard.AddRow().
		AddCallback(h.messages.VolunteerMenuProfileButton, schemes.DEFAULT, callbackMainMenuProfile)
	keyboard.AddRow().
		AddCallback(h.messages.VolunteerMenuMainButton, schemes.DEFAULT, callbackProfileBack)

	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) volunteerBackKeyboard() *maxbot.Keyboard {
	backLabel := h.messages.VolunteerMenuBackButton
	if strings.TrimSpace(backLabel) == "" {
		backLabel = "⬅️ Назад"
	}
	mainLabel := h.messages.VolunteerMenuMainButton
	if strings.TrimSpace(mainLabel) == "" {
		mainLabel = "🏠 Главное меню"
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(backLabel, schemes.DEFAULT, callbackVolunteerBack)
	keyboard.AddRow().
		AddCallback(mainLabel, schemes.DEFAULT, callbackProfileBack)

	return keyboard
}

func (h *MessageHandler) showProfileHistory(ctx context.Context, chatID, userID int64) {
	h.renderMenu(ctx, chatID, userID, h.messages.ProfileHistoryText, h.singleButtonKeyboard(h.messages.ProfileBackButton, callbackProfileBack))
}

func (h *MessageHandler) showProfileEdit(ctx context.Context, chatID, userID int64) {
	h.renderMenu(ctx, chatID, userID, h.messages.ProfileEditText, h.singleButtonKeyboard(h.messages.ProfileBackButton, callbackProfileBack))
}

func (h *MessageHandler) showProfileSecurity(ctx context.Context, chatID, userID int64) {
	text := fmt.Sprintf("%s\n\n%s", h.messages.ProfileSecurityTitle, fmt.Sprintf(h.messages.ProfileSecurityText, h.messages.ProfileSecuritySOSLink))

	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddLink(h.messages.ProfileSecuritySOSButton, schemes.POSITIVE, h.messages.ProfileSecuritySOSLink)
	keyboard.AddRow().
		AddCallback(h.messages.ProfileBackButton, schemes.DEFAULT, callbackProfileBack)

	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) buildProfileText(ctx context.Context, userID int64) (string, error) {
	if h.user == nil {
		return "", fmt.Errorf("user service client is not configured")
	}

	maxID := fmt.Sprintf("%d", userID)
	userResp, err := h.user.GetUserByMaxID(ctx, &userpb.GetUserByMaxIDRequest{MaxId: maxID})
	if err != nil {
		return "", err
	}
	if userResp.GetError() != nil {
		return "", fmt.Errorf("user service error: %s", userResp.GetError().GetMessage())
	}

	user := userResp.GetUser()
	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	var builder strings.Builder
	builder.WriteString(h.messages.ProfileTitle)
	builder.WriteString("\n\n")

	if name := strings.TrimSpace(user.GetName()); name != "" {
		builder.WriteString(fmt.Sprintf("*Имя:* %s\n", name))
	}
	if age := user.GetAge(); age > 0 {
		builder.WriteString(fmt.Sprintf("*Возраст:* %d\n", age))
	}
	if city := strings.TrimSpace(user.GetGeolocation()); city != "" {
		builder.WriteString(fmt.Sprintf("*Город:* %s\n", city))
	}

	builder.WriteString("\n")

	if about := strings.TrimSpace(user.GetAbout()); about != "" {
		builder.WriteString(h.messages.ProfileSkillsTitle)
		builder.WriteString("\n")

		for _, chunk := range strings.Split(about, ";") {
			if item := strings.TrimSpace(chunk); item != "" {
				builder.WriteString("• ")
				builder.WriteString(item)
				builder.WriteString("\n")
			}
		}

		builder.WriteString("\n")
	}

	level := "Новичок"
	if group := user.GetReputationGroup(); group != nil && strings.TrimSpace(group.GetName()) != "" {
		level = group.GetName()
	}

	balance := 0
	if balanceResp, err := h.user.GetBalance(ctx, &userpb.GetBalanceRequest{MaxId: maxID}); err != nil {
		h.logger.Warn("failed to fetch balance", zap.Error(err))
	} else if balanceResp.GetError() != nil {
		h.logger.Warn("balance response error", zap.String("message", balanceResp.GetError().GetMessage()))
	} else {
		balance = int(balanceResp.GetBalance())
	}

	builder.WriteString(fmt.Sprintf(h.messages.ProfileLevelBalanceTemplate, level, balance))

	return builder.String(), nil
}

func (h *MessageHandler) showProfileCoinsMenu(ctx context.Context, chatID, userID int64) {
	buttons := h.messages.CoinsButtons
	keyboard := h.api.Messages.NewKeyboardBuilder()

	if len(buttons) > 0 {
		row := keyboard.AddRow()
		row.AddCallback(buttons[0], schemes.DEFAULT, callbackCoinsHowToGet)
		if len(buttons) > 1 {
			row.AddCallback(buttons[1], schemes.DEFAULT, callbackCoinsHowToSpend)
		}
	}
	if len(buttons) > 2 {
		row := keyboard.AddRow()
		row.AddCallback(buttons[2], schemes.DEFAULT, callbackCoinsLevels)
		if len(buttons) > 3 {
			row.AddCallback(buttons[3], schemes.DEFAULT, callbackProfileBack)
		}
	}

	h.renderMenu(ctx, chatID, userID, h.messages.CoinsIntroText, keyboard)
}

func (h *MessageHandler) showCoinsHowToGet(ctx context.Context, chatID, userID int64) {
	h.renderMenu(ctx, chatID, userID, h.messages.CoinsHowToGetText, h.coinsDetailKeyboard())
}

func (h *MessageHandler) showCoinsHowToSpend(ctx context.Context, chatID, userID int64) {
	h.renderMenu(ctx, chatID, userID, h.messages.CoinsHowToSpendText, h.coinsDetailKeyboard())
}

func (h *MessageHandler) showCoinsLevels(ctx context.Context, chatID, userID int64) {
	h.renderMenu(ctx, chatID, userID, h.messages.CoinsLevelsText, h.coinsDetailKeyboard())
}

func (h *MessageHandler) showAboutDobrikaMenu(ctx context.Context, chatID, userID int64) {
	h.renderMenu(ctx, chatID, userID, h.messages.AboutDobrikaText, h.aboutMenuKeyboard())
}

func (h *MessageHandler) singleButtonKeyboard(text string, payload string) *maxbot.Keyboard {
	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(text, schemes.DEFAULT, payload)
	return keyboard
}

func (h *MessageHandler) coinsDetailKeyboard() *maxbot.Keyboard {
	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(h.messages.ProfileCoinsButton, schemes.DEFAULT, callbackProfileCoins)
	keyboard.AddRow().
		AddCallback(h.messages.CoinsBackButton, schemes.DEFAULT, callbackProfileBack)
	return keyboard
}

func (h *MessageHandler) aboutMenuKeyboard() *maxbot.Keyboard {
	buttons := h.messages.AboutDobrikaButtons
	keyboard := h.api.Messages.NewKeyboardBuilder()

	if len(buttons) > 0 {
		keyboard.AddRow().
			AddCallback(buttons[0], schemes.DEFAULT, callbackAboutHowItWorks)
	}
	if len(buttons) > 1 {
		keyboard.AddRow().
			AddCallback(buttons[1], schemes.DEFAULT, callbackAboutRules)
	}
	if len(buttons) > 2 {
		keyboard.AddRow().
			AddCallback(buttons[2], schemes.DEFAULT, callbackAboutInitiator)
	}
	if len(buttons) > 3 {
		keyboard.AddRow().
			AddCallback(buttons[3], schemes.DEFAULT, callbackAboutSupport)
	}
	if len(buttons) > 4 {
		keyboard.AddRow().
			AddCallback(buttons[4], schemes.DEFAULT, callbackAboutBack)
	}

	return keyboard
}
