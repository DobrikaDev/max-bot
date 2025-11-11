package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	taskpb "DobrikaDev/max-bot/internal/generated/taskpb"
	userpb "DobrikaDev/max-bot/internal/generated/userpb"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	schemes "github.com/max-messenger/max-bot-api-client-go/schemes"
	"go.uber.org/zap"
)

type taskCreationStep int

const (
	taskStepNone taskCreationStep = iota
	taskStepName
	taskStepDescription
	taskStepComplete
)

const (
	taskListPageSize   = 5
	searchQueryTypeGeo = "SGeoTasks"
	searchDefaultQuery = "geo"
)

type volunteerTasksViewMode string

const (
	volunteerTasksViewModeNone     volunteerTasksViewMode = ""
	volunteerTasksViewModeAll      volunteerTasksViewMode = "all"
	volunteerTasksViewModeOnDemand volunteerTasksViewMode = "on_demand"
)

type volunteerTasksFilter string

const (
	volunteerTasksFilterAll    volunteerTasksFilter = "all"
	volunteerTasksFilterReward volunteerTasksFilter = "reward"
	volunteerTasksFilterTeam   volunteerTasksFilter = "team"
	volunteerTasksFilterOnline volunteerTasksFilter = "online"
)

var errVolunteerLocationMissing = errors.New("volunteer location missing")

type volunteerTaskDisplayEntry struct {
	task   *taskpb.Task
	status string
	joined bool
	reward bool
	team   bool
	online bool
	order  int
}

type taskCreationSession struct {
	UserID      int64
	ChatID      int64
	MessageID   string
	CustomerID  string
	Name        string
	Description string
	Current     taskCreationStep
}

func (s *taskCreationSession) isInProgress() bool {
	return s != nil && s.Current != taskStepNone && s.Current != taskStepComplete
}

type taskSessionStore struct {
	mu       sync.RWMutex
	sessions map[int64]*taskCreationSession
}

type taskAssignment struct {
	UserID string
	Status string
}

type taskAssignmentJSON struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

func newTaskSessionStore() *taskSessionStore {
	return &taskSessionStore{sessions: make(map[int64]*taskCreationSession)}
}

func (s *taskSessionStore) get(userID int64) (*taskCreationSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[userID]
	return session, ok
}

func (s *taskSessionStore) upsert(session *taskCreationSession) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.UserID] = session
}

func (s *taskSessionStore) delete(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, userID)
}

func (h *MessageHandler) tryHandleTaskCreationMessage(ctx context.Context, update *schemes.MessageCreatedUpdate) bool {
	session, ok := h.taskSessions.get(update.Message.Sender.UserId)
	if !ok || !session.isInProgress() {
		return false
	}

	text := strings.TrimSpace(update.GetText())

	switch session.Current {
	case taskStepName:
		if text == "" {
			h.sendTaskSessionMessage(ctx, session, h.taskCreateNameRetryText(), emptyKeyboard())
			return true
		}
		session.Name = text
		session.Current = taskStepDescription
		h.taskSessions.upsert(session)
		h.promptTaskDescription(ctx, session)
	case taskStepDescription:
		if text == "" {
			h.sendTaskSessionMessage(ctx, session, h.taskCreateDescriptionRetryText(), emptyKeyboard())
			return true
		}
		session.Description = text
		session.Current = taskStepComplete
		h.taskSessions.upsert(session)
		h.finalizeTaskCreation(ctx, session)
	default:
		h.logger.Debug("task creation message in unexpected step", zap.Int("step", int(session.Current)))
	}

	return true
}

func (h *MessageHandler) handleCustomerManageTasks(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	if update.Message == nil {
		return
	}

	userID := update.Callback.User.UserId
	chatID := update.Message.Recipient.ChatId

	if h.task == nil {
		text := h.taskServiceUnavailableText()
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}

	customer, err := h.getCustomerByMaxID(ctx, fmt.Sprintf("%d", userID))
	if err != nil {
		h.logger.Error("failed to fetch customer for task list", zap.Error(err), zap.Int64("user_id", userID))
		text := h.taskFetchErrorText()
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}
	if customer == nil || strings.TrimSpace(customer.GetMaxId()) == "" {
		h.renderMenu(ctx, chatID, userID, h.taskCreateNoCustomerText(), h.customerBackKeyboard())
		return
	}

	h.showCustomerTasksMenu(ctx, chatID, userID, strings.TrimSpace(customer.GetMaxId()), 0)
}

func (h *MessageHandler) handleCustomerTasksPage(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, payload string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil {
		return
	}

	parts := strings.Split(payload, ":")
	if len(parts) != 2 {
		h.logger.Debug("invalid customer tasks page payload", zap.String("payload", payload))
		return
	}

	customerID := strings.TrimSpace(parts[0])
	page, err := strconv.Atoi(parts[1])
	if err != nil {
		h.logger.Warn("failed to parse customer tasks page", zap.Error(err), zap.String("payload", payload))
		page = 0
	}
	if page < 0 {
		page = 0
	}

	chatID := callbackQuery.Message.Recipient.ChatId
	userID := callbackQuery.Callback.User.UserId

	if customerID == "" {
		customerID = fmt.Sprintf("%d", userID)
	}

	h.showCustomerTasksMenu(ctx, chatID, userID, customerID, page)
}

func (h *MessageHandler) handleCustomerManageCreateTask(ctx context.Context, update *schemes.MessageCallbackUpdate) {
	if update.Message == nil {
		return
	}

	userID := update.Callback.User.UserId
	chatID := update.Message.Recipient.ChatId

	if h.task == nil {
		text := h.taskServiceUnavailableText()
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}

	customer, err := h.getCustomerByMaxID(ctx, fmt.Sprintf("%d", userID))
	if err != nil {
		h.logger.Error("failed to fetch customer for task creation", zap.Error(err), zap.Int64("user_id", userID))
		text := h.taskFetchErrorText()
		h.renderMenu(ctx, chatID, userID, text, h.customerBackKeyboard())
		return
	}
	if customer == nil || strings.TrimSpace(customer.GetMaxId()) == "" {
		h.renderMenu(ctx, chatID, userID, h.taskCreateNoCustomerText(), h.customerBackKeyboard())
		return
	}

	session := &taskCreationSession{
		UserID:     userID,
		ChatID:     chatID,
		CustomerID: strings.TrimSpace(customer.GetMaxId()),
		Current:    taskStepName,
		MessageID:  update.Message.Body.Mid,
	}

	h.startTaskCreationFlow(ctx, session)
}

func (h *MessageHandler) startTaskCreationFlow(ctx context.Context, session *taskCreationSession) {
	h.taskSessions.upsert(session)
	h.sendTaskSessionMessage(ctx, session, h.taskCreateNamePromptText(), emptyKeyboard())
}

func (h *MessageHandler) promptTaskDescription(ctx context.Context, session *taskCreationSession) {
	h.sendTaskSessionMessage(ctx, session, h.taskCreateDescriptionPromptText(), emptyKeyboard())
}

func (h *MessageHandler) finalizeTaskCreation(ctx context.Context, session *taskCreationSession) {
	if h.task == nil {
		h.sendTaskSessionMessage(ctx, session, h.taskServiceUnavailableText(), emptyKeyboard())
		h.taskSessions.delete(session.UserID)
		return
	}

	req := &taskpb.CreateTaskRequest{
		Task: &taskpb.Task{
			CustomerId:       session.CustomerID,
			Name:             strings.TrimSpace(session.Name),
			Description:      strings.TrimSpace(session.Description),
			VerificationType: taskpb.VerificationType_VERIFICATION_TYPE_NONE,
			Cost:             0,
			MembersCount:     1,
		},
	}

	resp, err := h.task.CreateTask(ctx, req)
	if err != nil {
		h.logger.Error("failed to create task", zap.Error(err), zap.String("customer_id", session.CustomerID))
		h.sendTaskSessionMessage(ctx, session, h.taskCreateErrorText(), h.customerBackKeyboard())
		h.taskSessions.delete(session.UserID)
		return
	}
	if resp.GetError() != nil {
		h.logger.Warn("task service returned error", zap.String("message", resp.GetError().GetMessage()))
		h.sendTaskSessionMessage(ctx, session, h.taskCreateErrorText(), h.customerBackKeyboard())
		h.taskSessions.delete(session.UserID)
		return
	}

	h.taskSessions.delete(session.UserID)

	success := h.taskCreateSuccessText(session.Name)
	h.showCustomerTasksMenu(ctx, session.ChatID, session.UserID, session.CustomerID, 0, success)
}

func (h *MessageHandler) sendTaskSessionMessage(ctx context.Context, session *taskCreationSession, text string, keyboard *maxbot.Keyboard) {
	messageID, err := h.sendInteractiveMessage(ctx, session.ChatID, session.UserID, text, keyboard)
	if err != nil {
		h.logger.Error("failed to send task session message", zap.Error(err), zap.Int64("chat_id", session.ChatID))
		return
	}

	session.MessageID = messageID
	h.taskSessions.upsert(session)
}

func (h *MessageHandler) showCustomerTasksMenu(ctx context.Context, chatID, userID int64, customerID string, page int, intro ...string) {
	text, keyboard := h.buildCustomerTasksView(ctx, customerID, page, intro...)
	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) buildCustomerTasksView(ctx context.Context, customerID string, page int, intro ...string) (string, *maxbot.Keyboard) {
	var builder strings.Builder
	if len(intro) > 0 && strings.TrimSpace(intro[0]) != "" {
		builder.WriteString(strings.TrimSpace(intro[0]))
		builder.WriteString("\n\n")
	}

	if page < 0 {
		page = 0
	}

	if h.task == nil {
		builder.WriteString(h.taskServiceUnavailableText())
		return builder.String(), h.customerBackKeyboard()
	}

	limit := int32(taskListPageSize)
	offset := int32(page * taskListPageSize)

	resp, err := h.task.GetTasks(ctx, &taskpb.GetTasksRequest{CustomerId: customerID, Limit: limit, Offset: offset})
	if err != nil {
		h.logger.Error("failed to get tasks", zap.Error(err), zap.String("customer_id", customerID))
		builder.WriteString(h.taskFetchErrorText())
		return builder.String(), h.customerBackKeyboard()
	}

	if resp.GetError() != nil {
		h.logger.Warn("task service returned error on list", zap.String("message", resp.GetError().GetMessage()))
		builder.WriteString(h.taskFetchErrorText())
		return builder.String(), h.customerBackKeyboard()
	}

	total := int(resp.GetTotal())
	tasks := resp.GetTasks()

	if len(tasks) == 0 && total > 0 && offset >= int32(total) && page > 0 {
		page = (total - 1) / taskListPageSize
		if page < 0 {
			page = 0
		}
		offset = int32(page * taskListPageSize)

		resp, err = h.task.GetTasks(ctx, &taskpb.GetTasksRequest{CustomerId: customerID, Limit: limit, Offset: offset})
		if err != nil {
			h.logger.Error("failed to get tasks after page adjustment", zap.Error(err), zap.String("customer_id", customerID))
			builder.WriteString(h.taskFetchErrorText())
			return builder.String(), h.customerBackKeyboard()
		}

		if resp.GetError() != nil {
			h.logger.Warn("task service returned error on adjusted list", zap.String("message", resp.GetError().GetMessage()))
			builder.WriteString(h.taskFetchErrorText())
			return builder.String(), h.customerBackKeyboard()
		}

		total = int(resp.GetTotal())
		tasks = resp.GetTasks()
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()

	if len(tasks) == 0 {
		builder.WriteString(h.customerTasksEmptyText())
	} else {
		title := strings.TrimSpace(h.messages.CustomerTasksListText)
		if title == "" {
			title = "Select a task using the buttons below."
		}
		builder.WriteString(title)
		builder.WriteString("\n")

		startIndex := int(offset)
		for idx, task := range tasks {
			name := strings.TrimSpace(task.GetName())
			if name == "" {
				name = "Без названия"
			}

			label := truncateLabel(fmt.Sprintf("%d. %s", startIndex+idx+1, name), 40)
			keyboard.AddRow().
				AddCallback(label, schemes.DEFAULT, fmt.Sprintf("%s:%s", callbackCustomerTaskView, task.GetId()))
		}

		if total > taskListPageSize {
			footerTemplate := strings.TrimSpace(h.messages.CustomerTasksPageFooter)
			if footerTemplate == "" {
				footerTemplate = "Страница %d из %d"
			}
			totalPages := 1
			if total > 0 {
				totalPages = (total + taskListPageSize - 1) / taskListPageSize
			}
			builder.WriteString("\n")
			builder.WriteString(fmt.Sprintf(footerTemplate, page+1, totalPages))
			builder.WriteString("\n")
		}
	}

	hasPrev := page > 0
	hasNext := false
	if total > 0 {
		hasNext = (page+1)*taskListPageSize < total
	} else if len(tasks) == taskListPageSize {
		hasNext = true
	}

	if len(tasks) > 0 && (hasPrev || hasNext) {
		prevLabel := strings.TrimSpace(h.messages.CustomerTasksPrevButton)
		if prevLabel == "" {
			prevLabel = "⬅️ Назад"
		}
		nextLabel := strings.TrimSpace(h.messages.CustomerTasksNextButton)
		if nextLabel == "" {
			nextLabel = "➡️ Далее"
		}
		row := keyboard.AddRow()
		if hasPrev {
			row.AddCallback(prevLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s:%d", callbackCustomerTasksPage, customerID, page-1))
		}
		if hasNext {
			row.AddCallback(nextLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s:%d", callbackCustomerTasksPage, customerID, page+1))
		}
	}

	createLabel := h.messages.CustomerManageCreateTaskButton
	if strings.TrimSpace(createLabel) == "" {
		createLabel = "Создать задачу"
	}
	backLabel := h.messages.CustomerManageBackButton
	if strings.TrimSpace(backLabel) == "" {
		backLabel = "⬅️ Назад"
	}

	keyboard.AddRow().
		AddCallback(createLabel, schemes.POSITIVE, callbackCustomerManageCreateTask)
	keyboard.AddRow().
		AddCallback(backLabel, schemes.DEFAULT, callbackCustomerManageBack)

	return builder.String(), keyboard
}

func (h *MessageHandler) showVolunteerTasksList(ctx context.Context, chatID, userID int64, mode volunteerTasksViewMode, filter volunteerTasksFilter, intro string, page int) {
	text, keyboard := h.buildVolunteerTasksView(ctx, userID, mode, filter, intro, page)
	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) buildVolunteerTasksView(ctx context.Context, userID int64, mode volunteerTasksViewMode, filter volunteerTasksFilter, intro string, page int) (string, *maxbot.Keyboard) {
	switch mode {
	case volunteerTasksViewModeOnDemand:
		return h.buildVolunteerOnDemandView(ctx, userID, intro, page)
	default:
		return h.buildVolunteerGeoTasksView(ctx, userID, filter, intro, page)
	}
}

func (h *MessageHandler) buildVolunteerOnDemandView(ctx context.Context, userID int64, intro string, page int) (string, *maxbot.Keyboard) {
	var builder strings.Builder
	displayIntro := strings.TrimSpace(intro)
	if displayIntro == "" {
		displayIntro = strings.TrimSpace(h.messages.VolunteerOnDemandPlaceholder)
	}
	if displayIntro != "" {
		builder.WriteString(displayIntro)
		builder.WriteString("\n\n")
	}

	if h.task == nil {
		builder.WriteString(h.volunteerTasksUnavailableText())
		return builder.String(), h.volunteerBackKeyboard()
	}

	if page < 0 {
		page = 0
	}

	limit := int32(taskListPageSize)
	offset := int32(page * taskListPageSize)

	resp, err := h.task.GetTasks(ctx, &taskpb.GetTasksRequest{Limit: limit, Offset: offset})
	if err != nil {
		h.logger.Error("failed to fetch tasks for volunteer list", zap.Error(err))
		builder.WriteString(h.volunteerTasksErrorText())
		return builder.String(), h.volunteerBackKeyboard()
	}

	if resp.GetError() != nil {
		h.logger.Warn("task service returned error for volunteer list", zap.String("message", resp.GetError().GetMessage()))
		builder.WriteString(h.volunteerTasksErrorText())
		return builder.String(), h.volunteerBackKeyboard()
	}

	total := int(resp.GetTotal())
	tasks := resp.GetTasks()

	if len(tasks) == 0 && total > 0 && offset >= int32(total) && page > 0 {
		page = (total - 1) / taskListPageSize
		if page < 0 {
			page = 0
		}
		offset = int32(page * taskListPageSize)

		resp, err = h.task.GetTasks(ctx, &taskpb.GetTasksRequest{Limit: limit, Offset: offset})
		if err != nil {
			h.logger.Error("failed to fetch tasks after page adjustment", zap.Error(err))
			builder.WriteString(h.volunteerTasksErrorText())
			return builder.String(), h.volunteerBackKeyboard()
		}

		if resp.GetError() != nil {
			h.logger.Warn("task service returned error for volunteer list after adjustment", zap.String("message", resp.GetError().GetMessage()))
			builder.WriteString(h.volunteerTasksErrorText())
			return builder.String(), h.volunteerBackKeyboard()
		}

		total = int(resp.GetTotal())
		tasks = resp.GetTasks()
	}

	if len(tasks) == 0 {
		builder.WriteString(h.volunteerOnDemandEmptyText())
		return builder.String(), h.volunteerBackKeyboard()
	}

	userIDStr := fmt.Sprintf("%d", userID)

	type taskEntry struct {
		task   *taskpb.Task
		status string
	}

	var (
		joined    []taskEntry
		available []taskEntry
	)

	for _, task := range tasks {
		assignments := parseTaskAssignments(task)
		status := assignmentStatusForUser(assignments, userIDStr)
		entry := taskEntry{task: task, status: status}
		if status == "" || isStatusRejected(status) {
			available = append(available, entry)
		} else {
			joined = append(joined, entry)
		}
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()
	baseIndex := int(offset)
	sectionIndex := baseIndex + 1

	if len(joined) > 0 {
		builder.WriteString(fmt.Sprintf("🌟 *Мои отклики:* %d\n", len(joined)))
		for _, entry := range joined {
			name := safeTaskName(entry.task.GetName())
			buttonLabel := truncateLabel(fmt.Sprintf("%d. %s %s", sectionIndex, name, volunteerStatusBadge(entry.status)), 45)
			keyboard.AddRow().
				AddCallback(buttonLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s", callbackVolunteerTaskView, entry.task.GetId()))
			sectionIndex++
		}
	}

	if len(available) > 0 {
		if sectionIndex > baseIndex+1 {
			builder.WriteString("────────────────────\n")
		}
		builder.WriteString(fmt.Sprintf("📋 *Свободные дела:* %d\n", len(available)))
		for _, entry := range available {
			name := safeTaskName(entry.task.GetName())
			buttonLabel := truncateLabel(fmt.Sprintf("%d. %s %s", sectionIndex, name, volunteerStatusBadge(entry.status)), 45)
			keyboard.AddRow().
				AddCallback(buttonLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s", callbackVolunteerTaskView, entry.task.GetId()))
			sectionIndex++
		}
	}

	if total > taskListPageSize {
		footer := strings.TrimSpace(h.messages.VolunteerTasksPageFooter)
		if footer == "" {
			footer = "Страница %d из %d"
		}
		totalPages := (total + taskListPageSize - 1) / taskListPageSize
		if totalPages < 1 {
			totalPages = 1
		}
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf(footer, page+1, totalPages))
		builder.WriteString("\n")
	}

	hasPrev := page > 0
	hasNext := false
	if total > 0 {
		hasNext = (page+1)*taskListPageSize < total
	} else if len(tasks) == taskListPageSize {
		hasNext = true
	}

	if len(tasks) > 0 && (hasPrev || hasNext) {
		prevLabel := strings.TrimSpace(h.messages.VolunteerTasksPrevButton)
		if prevLabel == "" {
			prevLabel = "⬅️ Назад"
		}
		nextLabel := strings.TrimSpace(h.messages.VolunteerTasksNextButton)
		if nextLabel == "" {
			nextLabel = "➡️ Далее"
		}
		row := keyboard.AddRow()
		if hasPrev {
			row.AddCallback(prevLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s:%s:%d", callbackVolunteerTasksPage, volunteerTasksViewModeOnDemand, volunteerTasksFilterAll, page-1))
		}
		if hasNext {
			row.AddCallback(nextLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s:%s:%d", callbackVolunteerTasksPage, volunteerTasksViewModeOnDemand, volunteerTasksFilterAll, page+1))
		}
	}

	keyboard.AddRow().
		AddCallback(h.messages.VolunteerMenuBackButton, schemes.DEFAULT, callbackVolunteerBack)

	return builder.String(), keyboard
}

func (h *MessageHandler) buildVolunteerGeoTasksView(ctx context.Context, userID int64, filter volunteerTasksFilter, intro string, page int) (string, *maxbot.Keyboard) {
	var builder strings.Builder
	displayIntro := strings.TrimSpace(intro)
	if displayIntro == "" {
		displayIntro = strings.TrimSpace(h.messages.VolunteerTasksPlaceholder)
	}
	if displayIntro != "" {
		builder.WriteString(displayIntro)
		builder.WriteString("\n\n")
	}

	if h.task == nil {
		builder.WriteString(h.volunteerTasksUnavailableText())
		return builder.String(), h.volunteerBackKeyboard()
	}

	tasks, err := h.fetchGeoTasks(ctx, userID)
	if err != nil {
		if errors.Is(err, errVolunteerLocationMissing) {
			builder.WriteString(h.volunteerLocationMissingText())
			keyboard := h.api.Messages.NewKeyboardBuilder()
			keyboard.AddRow().
				AddCallback(h.volunteerLocationUpdateButton(), schemes.POSITIVE, callbackMainMenuProfile)
			keyboard.AddRow().
				AddCallback(h.messages.VolunteerMenuBackButton, schemes.DEFAULT, callbackVolunteerBack)
			return builder.String(), keyboard
		}
		builder.WriteString(h.volunteerTasksErrorText())
		return builder.String(), h.volunteerBackKeyboard()
	}

	filtered := h.filterVolunteerTasks(tasks, userID, filter)
	if len(filtered) == 0 {
		builder.WriteString(h.volunteerFilterEmptyText())
		keyboard := h.api.Messages.NewKeyboardBuilder()
		h.appendVolunteerFilterRows(keyboard, volunteerTasksViewModeAll, filter)
		keyboard.AddRow().
			AddCallback(h.messages.VolunteerMenuBackButton, schemes.DEFAULT, callbackVolunteerBack)
		return builder.String(), keyboard
	}

	joinedCount := 0
	for _, entry := range filtered {
		if entry.joined {
			joinedCount++
		}
	}

	total := len(filtered)
	totalPages := (total + taskListPageSize - 1) / taskListPageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * taskListPageSize
	end := start + taskListPageSize
	if end > total {
		end = total
	}

	builder.WriteString(fmt.Sprintf("Найдено дел рядом: %d\n", total))
	if filter != volunteerTasksFilterAll {
		builder.WriteString(fmt.Sprintf("Фильтр: %s\n", h.currentFilterLabel(filter)))
	}
	if joinedCount > 0 {
		builder.WriteString(fmt.Sprintf("🌟 Твои отклики: %d\n", joinedCount))
	}
	builder.WriteString("\nВыбери дело ниже:\n")

	keyboard := h.api.Messages.NewKeyboardBuilder()
	h.appendVolunteerFilterRows(keyboard, volunteerTasksViewModeAll, filter)

	sectionIndex := start + 1
	for _, entry := range filtered[start:end] {
		name := safeTaskName(entry.task.GetName())
		badge := volunteerStatusBadge(entry.status)
		extra := h.taskFilterBadge(entry)
		buttonLabel := truncateLabel(fmt.Sprintf("%d. %s %s%s", sectionIndex, name, badge, extra), 45)
		keyboard.AddRow().
			AddCallback(buttonLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s", callbackVolunteerTaskView, entry.task.GetId()))
		sectionIndex++
	}

	if totalPages > 1 {
		builder.WriteString("\n")
		footerTemplate := strings.TrimSpace(h.messages.VolunteerTasksPageFooter)
		if footerTemplate == "" {
			footerTemplate = "Страница %d из %d"
		}
		builder.WriteString(fmt.Sprintf(footerTemplate, page+1, totalPages))
		builder.WriteString("\n")

		row := keyboard.AddRow()
		prevLabel := strings.TrimSpace(h.messages.VolunteerTasksPrevButton)
		if prevLabel == "" {
			prevLabel = "⬅️ Назад"
		}
		nextLabel := strings.TrimSpace(h.messages.VolunteerTasksNextButton)
		if nextLabel == "" {
			nextLabel = "➡️ Далее"
		}
		if page > 0 {
			row.AddCallback(prevLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s:%s:%d", callbackVolunteerTasksPage, volunteerTasksViewModeAll, filter, page-1))
		}
		if page < totalPages-1 {
			row.AddCallback(nextLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s:%s:%d", callbackVolunteerTasksPage, volunteerTasksViewModeAll, filter, page+1))
		}
	}

	keyboard.AddRow().
		AddCallback(h.messages.VolunteerMenuBackButton, schemes.DEFAULT, callbackVolunteerBack)

	return builder.String(), keyboard
}

func (h *MessageHandler) fetchGeoTasks(ctx context.Context, userID int64) ([]*taskpb.Task, error) {
	if h.user == nil {
		h.logger.Error("user service client is not configured for geo tasks", zap.Int64("user_id", userID))
		return nil, errVolunteerLocationMissing
	}

	maxID := fmt.Sprintf("%d", userID)
	userResp, err := h.user.GetUserByMaxID(ctx, &userpb.GetUserByMaxIDRequest{MaxId: maxID})
	if err != nil {
		h.logger.Error("failed to fetch user for geo tasks", zap.Error(err), zap.Int64("user_id", userID))
		return nil, err
	}
	if svcErr := userResp.GetError(); svcErr != nil {
		h.logger.Warn("user service returned error for geo tasks", zap.String("message", svcErr.GetMessage()), zap.Int64("user_id", userID))
		return nil, errVolunteerLocationMissing
	}

	user := userResp.GetUser()
	if user == nil {
		return nil, errVolunteerLocationMissing
	}

	geoData, ok := normalizeGeoCoordinates(user.GetGeolocation())
	if !ok {
		return nil, errVolunteerLocationMissing
	}

	req := &taskpb.SearchTasksRequest{
		Query:     searchDefaultQuery,
		QueryType: searchQueryTypeGeo,
		GeoData:   geoData,
	}

	resp, err := h.task.SearchTasks(ctx, req)
	if err != nil {
		h.logger.Error("failed to search geo tasks", zap.Error(err), zap.Int64("user_id", userID))
		return nil, err
	}

	if resp.GetError() != nil {
		h.logger.Warn("search tasks returned error", zap.String("message", resp.GetError().GetMessage()))
		return nil, fmt.Errorf("search tasks error: %s", resp.GetError().GetMessage())
	}

	return resp.GetTasks(), nil
}

func (h *MessageHandler) filterVolunteerTasks(tasks []*taskpb.Task, userID int64, filter volunteerTasksFilter) []volunteerTaskDisplayEntry {
	if len(tasks) == 0 {
		return nil
	}

	userIDStr := fmt.Sprintf("%d", userID)
	result := make([]volunteerTaskDisplayEntry, 0, len(tasks))

	for idx, task := range tasks {
		if task == nil {
			continue
		}

		assignments := parseTaskAssignments(task)
		status := assignmentStatusForUser(assignments, userIDStr)
		entry := volunteerTaskDisplayEntry{
			task:   task,
			status: status,
			joined: status != "" && !isStatusRejected(status),
			reward: task.GetCost() > 0,
			team:   task.GetMembersCount() > 1,
			online: isOnlineTask(task),
			order:  idx,
		}

		if passesVolunteerFilter(entry, filter) {
			result = append(result, entry)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].joined != result[j].joined {
			return result[i].joined
		}
		return result[i].order < result[j].order
	})

	return result
}

func passesVolunteerFilter(entry volunteerTaskDisplayEntry, filter volunteerTasksFilter) bool {
	switch filter {
	case volunteerTasksFilterReward:
		return entry.reward
	case volunteerTasksFilterTeam:
		return entry.team
	case volunteerTasksFilterOnline:
		return entry.online
	default:
		return true
	}
}

func (h *MessageHandler) appendVolunteerFilterRows(keyboard *maxbot.Keyboard, mode volunteerTasksViewMode, current volunteerTasksFilter) {
	if keyboard == nil {
		return
	}

	filters := []volunteerTasksFilter{
		volunteerTasksFilterAll,
		volunteerTasksFilterReward,
		volunteerTasksFilterTeam,
		volunteerTasksFilterOnline,
	}

	var row *maxbot.KeyboardRow
	for idx, f := range filters {
		if idx%2 == 0 {
			row = keyboard.AddRow()
		}
		if row == nil {
			continue
		}
		label := h.filterButtonLabel(f, current)
		scheme := schemes.DEFAULT
		if f == current {
			scheme = schemes.POSITIVE
		}
		row.AddCallback(label, scheme, fmt.Sprintf("%s:%s:%s", callbackVolunteerTasksFilter, mode, f))
	}
}

func (h *MessageHandler) filterButtonLabel(filter, current volunteerTasksFilter) string {
	label := strings.TrimSpace(h.filterBaseLabel(filter))
	if label == "" {
		switch filter {
		case volunteerTasksFilterReward:
			label = "💰 Награда"
		case volunteerTasksFilterTeam:
			label = "👥 Команда"
		case volunteerTasksFilterOnline:
			label = "💻 Онлайн"
		default:
			label = "📍 Рядом"
		}
	}
	if filter == current {
		if strings.HasPrefix(label, "✅") {
			return label
		}
		return "✅ " + label
	}
	return label
}

func (h *MessageHandler) filterBaseLabel(filter volunteerTasksFilter) string {
	switch filter {
	case volunteerTasksFilterReward:
		return h.messages.VolunteerTasksFilterRewardButton
	case volunteerTasksFilterTeam:
		return h.messages.VolunteerTasksFilterTeamButton
	case volunteerTasksFilterOnline:
		return h.messages.VolunteerTasksFilterOnlineButton
	default:
		return h.messages.VolunteerTasksFilterAllButton
	}
}

func (h *MessageHandler) currentFilterLabel(filter volunteerTasksFilter) string {
	switch filter {
	case volunteerTasksFilterReward:
		if text := strings.TrimSpace(h.messages.VolunteerTasksFilterRewardLabel); text != "" {
			return text
		}
		return "с наградой"
	case volunteerTasksFilterTeam:
		if text := strings.TrimSpace(h.messages.VolunteerTasksFilterTeamLabel); text != "" {
			return text
		}
		return "для команды"
	case volunteerTasksFilterOnline:
		if text := strings.TrimSpace(h.messages.VolunteerTasksFilterOnlineLabel); text != "" {
			return text
		}
		return "онлайн"
	default:
		if text := strings.TrimSpace(h.messages.VolunteerTasksFilterAllLabel); text != "" {
			return text
		}
		return "все рядом"
	}
}

func (h *MessageHandler) taskFilterBadge(entry volunteerTaskDisplayEntry) string {
	var parts []string
	if entry.reward {
		parts = append(parts, "💰")
	}
	if entry.team {
		parts = append(parts, "👥")
	}
	if entry.online {
		parts = append(parts, "💻")
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, "")
}

func (h *MessageHandler) volunteerFilterEmptyText() string {
	if text := strings.TrimSpace(h.messages.VolunteerTasksFilterEmptyText); text != "" {
		return text
	}
	return "По выбранному фильтру ничего не нашлось. Попробуй другой вариант 💚"
}

func (h *MessageHandler) volunteerLocationMissingText() string {
	if text := strings.TrimSpace(h.messages.VolunteerTasksLocationMissingText); text != "" {
		return text
	}
	return "Чтобы увидеть добрые дела рядом, обнови свою локацию в профиле 💚"
}

func (h *MessageHandler) volunteerLocationUpdateButton() string {
	if text := strings.TrimSpace(h.messages.VolunteerTasksLocationUpdateButton); text != "" {
		return text
	}
	return "📍 Обновить локацию"
}

func parseVolunteerTasksFilter(value string) volunteerTasksFilter {
	switch volunteerTasksFilter(strings.TrimSpace(value)) {
	case volunteerTasksFilterReward:
		return volunteerTasksFilterReward
	case volunteerTasksFilterTeam:
		return volunteerTasksFilterTeam
	case volunteerTasksFilterOnline:
		return volunteerTasksFilterOnline
	default:
		return volunteerTasksFilterAll
	}
}

func normalizeGeoCoordinates(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return "", false
	}

	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil || lat < -90 || lat > 90 {
		return "", false
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil || lon < -180 || lon > 180 {
		return "", false
	}

	return fmt.Sprintf("%.6f,%.6f", lat, lon), true
}

func isOnlineTask(task *taskpb.Task) bool {
	if task == nil {
		return false
	}

	for _, meta := range task.GetMeta() {
		if meta == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(meta.GetKey()))
		value := strings.ToLower(strings.TrimSpace(meta.GetValue()))

		switch key {
		case "task_type", "type":
			switch value {
			case "tt_onlinetask", "online", "remote":
				return true
			case "tt_offlinetask", "offline":
				return false
			}
		case "online":
			if value == "true" || value == "1" || value == "yes" {
				return true
			}
		}
	}

	return false
}

func (h *MessageHandler) taskCreateNamePromptText() string {
	if text := strings.TrimSpace(h.messages.TaskCreateNamePrompt); text != "" {
		return text
	}
	return "Как назовём доброе дело?"
}

func (h *MessageHandler) taskCreateDescriptionPromptText() string {
	if text := strings.TrimSpace(h.messages.TaskCreateDescriptionPrompt); text != "" {
		return text
	}
	return "Расскажи, что нужно сделать. Это поможет волонтёрам понять задачу."
}

func (h *MessageHandler) taskCreateNameRetryText() string {
	if text := strings.TrimSpace(h.messages.TaskCreateNameRetryText); text != "" {
		return text
	}
	return "Введите название доброго дела, пожалуйста."
}

func (h *MessageHandler) taskCreateDescriptionRetryText() string {
	if text := strings.TrimSpace(h.messages.TaskCreateDescriptionRetryText); text != "" {
		return text
	}
	return "Добавьте описание, чтобы волонтёры понимали, чем помочь."
}

func (h *MessageHandler) taskCreateSuccessText(name string) string {
	if text := strings.TrimSpace(h.messages.TaskCreateSuccessText); text != "" {
		if strings.Contains(text, "%s") {
			return fmt.Sprintf(text, strings.TrimSpace(name))
		}
		return text
	}
	return fmt.Sprintf("Доброе дело «%s» создано 💚", strings.TrimSpace(name))
}

func (h *MessageHandler) taskCreateErrorText() string {
	if text := strings.TrimSpace(h.messages.TaskCreateErrorText); text != "" {
		return text
	}
	return "Не удалось создать задачу. Попробуйте позже."
}

func (h *MessageHandler) taskServiceUnavailableText() string {
	if text := strings.TrimSpace(h.messages.TaskServiceUnavailableText); text != "" {
		return text
	}
	return "Сервис задач недоступен. Попробуйте позже."
}

func (h *MessageHandler) taskFetchErrorText() string {
	if text := strings.TrimSpace(h.messages.TaskFetchErrorText); text != "" {
		return text
	}
	return "Не удалось получить список задач. Попробуйте позже."
}

func (h *MessageHandler) taskCreateNoCustomerText() string {
	if text := strings.TrimSpace(h.messages.TaskCreateNoCustomerText); text != "" {
		return text
	}
	return "Сначала заполни профиль заказчика, чтобы создавать добрые дела."
}

func (h *MessageHandler) customerTasksEmptyText() string {
	if text := strings.TrimSpace(h.messages.CustomerTasksEmptyText); text != "" {
		return text
	}
	return "Пока задач нет. Создайте первое доброе дело!"
}

func (h *MessageHandler) customerTaskItemTemplate() string {
	if text := strings.TrimSpace(h.messages.CustomerTaskItemTemplate); text != "" {
		return text
	}
	return "• *%s*\n%s"
}

func (h *MessageHandler) customerTaskRewardDescription(taskName string) string {
	name := strings.TrimSpace(taskName)
	if text := strings.TrimSpace(h.messages.CustomerTaskRewardDescription); text != "" {
		if strings.Contains(text, "%s") {
			return fmt.Sprintf(text, name)
		}
		return text
	}
	if name == "" {
		return "Награда за выполнение задачи"
	}
	return fmt.Sprintf("Награда за выполнение задачи «%s»", name)
}

func (h *MessageHandler) customerTaskApproveSuccessText(taskName string, amount int32) string {
	name := strings.TrimSpace(taskName)
	text := strings.TrimSpace(h.messages.CustomerTaskApproveSuccessText)
	if text == "" {
		if amount > 0 {
			return fmt.Sprintf("Выполнение задачи подтверждено 💚\nНаграда %d добриков отправлена исполнителю.", amount)
		}
		return "Выполнение задачи подтверждено 💚"
	}

	switch {
	case strings.Contains(text, "%d") && strings.Contains(text, "%s"):
		return fmt.Sprintf(text, amount, name)
	case strings.Contains(text, "%d"):
		return fmt.Sprintf(text, amount)
	case strings.Contains(text, "%s"):
		return fmt.Sprintf(text, name)
	default:
		return text
	}
}

func (h *MessageHandler) volunteerTaskRewardNotification(taskName string, amount int32) string {
	name := strings.TrimSpace(taskName)
	text := strings.TrimSpace(h.messages.VolunteerTaskRewardNotification)
	if text == "" {
		return fmt.Sprintf("Ты получил(а) %d добриков за доброе дело «%s» 💚", amount, name)
	}

	switch {
	case strings.Contains(text, "%d") && strings.Contains(text, "%s"):
		return fmt.Sprintf(text, amount, name)
	case strings.Contains(text, "%d"):
		return fmt.Sprintf(text, amount)
	case strings.Contains(text, "%s"):
		return fmt.Sprintf(text, name)
	default:
		return text
	}
}

func (h *MessageHandler) volunteerTasksUnavailableText() string {
	if text := strings.TrimSpace(h.messages.VolunteerTasksUnavailableText); text != "" {
		return text
	}
	return "Сервис задач недоступен. Попробуйте позже."
}

func (h *MessageHandler) volunteerTasksErrorText() string {
	if text := strings.TrimSpace(h.messages.VolunteerTasksErrorText); text != "" {
		return text
	}
	return "Не получилось получить добрые дела. Попробуйте позже."
}

func (h *MessageHandler) volunteerTasksEmptyText() string {
	if text := strings.TrimSpace(h.messages.VolunteerTasksEmptyText); text != "" {
		return text
	}
	return "Сейчас нет активных задач. Загляните позже!"
}

func (h *MessageHandler) volunteerOnDemandEmptyText() string {
	if text := strings.TrimSpace(h.messages.VolunteerOnDemandEmptyText); text != "" {
		return text
	}
	return "У тебя пока нет активных откликов."
}

func (h *MessageHandler) volunteerTaskItemTemplate() string {
	if text := strings.TrimSpace(h.messages.VolunteerTaskItemTemplate); text != "" {
		return text
	}
	return "• *%s*\n%s"
}

func (h *MessageHandler) handleVolunteerTasksPage(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, payload string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil {
		return
	}

	parts := strings.Split(payload, ":")
	if len(parts) < 2 {
		h.logger.Debug("invalid volunteer tasks page payload", zap.String("payload", payload))
		return
	}

	mode := volunteerTasksViewMode(strings.TrimSpace(parts[0]))
	switch mode {
	case volunteerTasksViewModeAll, volunteerTasksViewModeOnDemand:
	default:
		mode = volunteerTasksViewModeAll
	}

	filter := volunteerTasksFilterAll
	pagePart := parts[len(parts)-1]
	if len(parts) >= 3 {
		filter = parseVolunteerTasksFilter(parts[1])
	}

	page, err := strconv.Atoi(pagePart)
	if err != nil {
		h.logger.Warn("failed to parse volunteer tasks page", zap.Error(err), zap.String("payload", payload))
		page = 0
	}
	if page < 0 {
		page = 0
	}

	chatID := callbackQuery.Message.Recipient.ChatId
	userID := callbackQuery.Callback.User.UserId

	h.showVolunteerTasksList(ctx, chatID, userID, mode, filter, "", page)
}

func (h *MessageHandler) handleVolunteerTasksFilter(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, payload string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil {
		return
	}

	parts := strings.Split(payload, ":")
	if len(parts) != 2 {
		h.logger.Debug("invalid volunteer tasks filter payload", zap.String("payload", payload))
		return
	}

	mode := volunteerTasksViewMode(strings.TrimSpace(parts[0]))
	switch mode {
	case volunteerTasksViewModeAll, volunteerTasksViewModeOnDemand:
	default:
		mode = volunteerTasksViewModeAll
	}

	filter := parseVolunteerTasksFilter(parts[1])

	chatID := callbackQuery.Message.Recipient.ChatId
	userID := callbackQuery.Callback.User.UserId

	h.showVolunteerTasksList(ctx, chatID, userID, mode, filter, "", 0)
}

func (h *MessageHandler) handleVolunteerTaskView(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, taskID string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || taskID == "" {
		return
	}

	chatID := callbackQuery.Message.Recipient.ChatId
	userID := callbackQuery.Callback.User.UserId

	h.showVolunteerTaskDetail(ctx, chatID, userID, taskID)
}

func (h *MessageHandler) handleVolunteerTaskJoin(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, taskID string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || taskID == "" {
		return
	}

	if h.task == nil {
		h.showVolunteerTasksList(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, volunteerTasksViewModeNone, volunteerTasksFilterAll, h.volunteerTasksUnavailableText(), 0)
		return
	}

	userID := fmt.Sprintf("%d", callbackQuery.Callback.User.UserId)
	chatID := callbackQuery.Message.Recipient.ChatId

	resp, err := h.task.UserJoinTask(ctx, &taskpb.UserJoinTaskRequest{UserId: userID, TaskId: taskID})
	if err != nil || (resp != nil && resp.GetError() != nil) {
		h.showVolunteerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.VolunteerTaskJoinErrorText)
		return
	}

	h.showVolunteerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.VolunteerTaskJoinSuccessText)
}

func (h *MessageHandler) handleVolunteerTaskLeave(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, taskID string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || taskID == "" {
		return
	}

	if h.task == nil {
		h.showVolunteerTasksList(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, volunteerTasksViewModeNone, volunteerTasksFilterAll, h.volunteerTasksUnavailableText(), 0)
		return
	}

	userID := fmt.Sprintf("%d", callbackQuery.Callback.User.UserId)
	chatID := callbackQuery.Message.Recipient.ChatId

	resp, err := h.task.UserLeaveTask(ctx, &taskpb.UserLeaveTaskRequest{UserId: userID, TaskId: taskID})
	if err != nil || (resp != nil && resp.GetError() != nil) {
		h.showVolunteerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.VolunteerTaskLeaveErrorText)
		return
	}

	h.showVolunteerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.VolunteerTaskLeaveSuccessText)
}

func (h *MessageHandler) handleVolunteerTaskConfirm(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, taskID string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || taskID == "" {
		return
	}

	if h.task == nil {
		h.showVolunteerTasksList(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, volunteerTasksViewModeNone, volunteerTasksFilterAll, h.volunteerTasksUnavailableText(), 0)
		return
	}

	userID := fmt.Sprintf("%d", callbackQuery.Callback.User.UserId)
	chatID := callbackQuery.Message.Recipient.ChatId

	resp, err := h.task.UserConfirmTask(ctx, &taskpb.UserConfirmTaskRequest{UserId: userID, TaskId: taskID})
	if err != nil || (resp != nil && resp.GetError() != nil) {
		h.showVolunteerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.VolunteerTaskConfirmErrorText)
		return
	}

	h.showVolunteerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.VolunteerTaskConfirmSuccessText)
}

func (h *MessageHandler) showVolunteerTaskDetail(ctx context.Context, chatID, userID int64, taskID string, intro ...string) {
	if h.task == nil {
		h.showVolunteerTasksList(ctx, chatID, userID, volunteerTasksViewModeNone, volunteerTasksFilterAll, h.volunteerTasksUnavailableText(), 0)
		return
	}

	task, err := h.getTaskByID(ctx, taskID)
	if err != nil || task == nil {
		h.logger.Error("failed to fetch task detail", zap.Error(err), zap.String("task_id", taskID))
		h.showVolunteerTasksList(ctx, chatID, userID, volunteerTasksViewModeNone, volunteerTasksFilterAll, h.volunteerTasksErrorText(), 0)
		return
	}

	var builder strings.Builder
	if len(intro) > 0 && strings.TrimSpace(intro[0]) != "" {
		builder.WriteString(strings.TrimSpace(intro[0]))
		builder.WriteString("\n\n")
	}

	title := h.messages.VolunteerTaskDetailTitle
	if strings.TrimSpace(title) == "" {
		title = "*%s*"
	}

	name := safeTaskName(task.GetName())
	builder.WriteString(fmt.Sprintf(title, name))
	builder.WriteString("\n\n")

	builder.WriteString(safeTaskDescription(task.GetDescription()))

	assignments := parseTaskAssignments(task)
	userIDStr := fmt.Sprintf("%d", userID)
	status := assignmentStatusForUser(assignments, userIDStr)
	if statusLabel := volunteerStatusLabel(status); statusLabel != "" {
		builder.WriteString("\n\n*Статус:* ")
		builder.WriteString(statusLabel)
	}

	keyboard := h.api.Messages.NewKeyboardBuilder()

	joinLabel := h.messages.VolunteerTaskJoinButton
	if strings.TrimSpace(joinLabel) == "" {
		joinLabel = "Откликнуться"
	}
	leaveLabel := h.messages.VolunteerTaskLeaveButton
	if strings.TrimSpace(leaveLabel) == "" {
		leaveLabel = "Отказаться"
	}
	confirmLabel := h.messages.VolunteerTaskConfirmButton
	if strings.TrimSpace(confirmLabel) == "" {
		confirmLabel = "Я помог(ла)"
	}
	backLabel := h.messages.VolunteerTaskDetailBackButton
	if strings.TrimSpace(backLabel) == "" {
		backLabel = "⬅️ К списку"
	}

	if allowVolunteerJoin(status) {
		keyboard.AddRow().
			AddCallback(joinLabel, schemes.POSITIVE, fmt.Sprintf("%s:%s", callbackVolunteerTaskJoin, taskID))
	}

	if allowVolunteerLeave(status) {
		keyboard.AddRow().
			AddCallback(leaveLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s", callbackVolunteerTaskLeave, taskID))
	}

	if allowVolunteerConfirm(status) {
		keyboard.AddRow().
			AddCallback(confirmLabel, schemes.POSITIVE, fmt.Sprintf("%s:%s", callbackVolunteerTaskConfirm, taskID))
	}

	keyboard.AddRow().
		AddCallback(backLabel, schemes.DEFAULT, callbackVolunteerTasks)
	keyboard.AddRow().
		AddCallback(h.messages.VolunteerMenuMainButton, schemes.DEFAULT, callbackVolunteerBack)

	h.renderMenu(ctx, chatID, userID, builder.String(), keyboard)
}

func (h *MessageHandler) handleCustomerTaskView(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, taskID string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || taskID == "" {
		return
	}

	chatID := callbackQuery.Message.Recipient.ChatId
	userID := callbackQuery.Callback.User.UserId

	h.showCustomerTaskDetail(ctx, chatID, userID, taskID)
}

func (h *MessageHandler) handleCustomerTaskApprove(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, data string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || data == "" {
		return
	}

	taskID, volunteerID, ok := splitTaskAssignmentData(data)
	if !ok {
		return
	}

	if h.task == nil {
		h.renderMenu(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, h.taskServiceUnavailableText(), h.customerBackKeyboard())
		return
	}

	task, err := h.getTaskByID(ctx, taskID)
	if err != nil || task == nil {
		if err != nil {
			h.logger.Error("failed to fetch task before approval reward", zap.Error(err), zap.String("task_id", taskID))
		} else {
			h.logger.Warn("task not found before approval reward", zap.String("task_id", taskID))
		}
		h.showCustomerTaskAssignmentDetail(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, taskID, volunteerID, h.messages.CustomerTaskDecisionErrorText)
		return
	}

	chatID := callbackQuery.Message.Recipient.ChatId

	resp, err := h.task.ApproveTask(ctx, &taskpb.ApproveTaskRequest{UserId: volunteerID, TaskId: taskID})
	if err != nil || (resp != nil && resp.GetError() != nil) {
		h.showCustomerTaskAssignmentDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, volunteerID, h.messages.CustomerTaskDecisionErrorText)
		return
	}

	successText := h.customerTaskApproveSuccessText(task.GetName(), task.GetCost())

	if cost := task.GetCost(); cost > 0 {
		if h.user == nil {
			h.logger.Error("user service client is not configured for reward credit", zap.String("task_id", taskID), zap.String("volunteer_id", volunteerID))
			h.showCustomerTaskAssignmentDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, volunteerID, h.messages.CustomerTaskDecisionErrorText)
			return
		}

		opReq := &userpb.CreateOperationRequest{
			MaxId:       volunteerID,
			Amount:      cost,
			Type:        userpb.BalanceOperationType_BALANCE_OPERATION_TYPE_DEPOSIT,
			Description: h.customerTaskRewardDescription(task.GetName()),
		}

		opResp, err := h.user.CreateOperation(ctx, opReq)
		if err != nil {
			h.logger.Error("failed to credit volunteer reward", zap.Error(err), zap.String("task_id", taskID), zap.String("volunteer_id", volunteerID))
			h.showCustomerTaskAssignmentDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, volunteerID, h.messages.CustomerTaskDecisionErrorText)
			return
		}

		if svcErr := opResp.GetError(); svcErr != nil {
			h.logger.Warn("user service returned error when crediting reward", zap.String("task_id", taskID), zap.String("volunteer_id", volunteerID), zap.String("message", svcErr.GetMessage()))
			h.showCustomerTaskAssignmentDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, volunteerID, h.messages.CustomerTaskDecisionErrorText)
			return
		}

		volunteerNumericID, err := strconv.ParseInt(volunteerID, 10, 64)
		if err != nil || volunteerNumericID <= 0 {
			h.logger.Warn("failed to parse volunteer id for reward notification", zap.String("volunteer_id", volunteerID), zap.Error(err))
		} else {
			notification := strings.TrimSpace(h.volunteerTaskRewardNotification(task.GetName(), cost))
			if notification != "" {
				if _, err := h.sendInteractiveMessage(ctx, volunteerNumericID, volunteerNumericID, notification, nil); err != nil {
					h.logger.Error("failed to send reward notification", zap.Error(err), zap.Int64("volunteer_id", volunteerNumericID), zap.String("task_id", taskID))
				}
			}
		}
	}

	h.showCustomerTaskAssignmentDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, volunteerID, successText)
}

func (h *MessageHandler) handleCustomerTaskReject(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, data string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || data == "" {
		return
	}

	taskID, volunteerID, ok := splitTaskAssignmentData(data)
	if !ok {
		return
	}

	if h.task == nil {
		h.renderMenu(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, h.taskServiceUnavailableText(), h.customerBackKeyboard())
		return
	}

	chatID := callbackQuery.Message.Recipient.ChatId

	resp, err := h.task.RejectTask(ctx, &taskpb.RejectTaskRequest{UserId: volunteerID, TaskId: taskID})
	if err != nil || (resp != nil && resp.GetError() != nil) {
		h.showCustomerTaskAssignmentDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, volunteerID, h.messages.CustomerTaskDecisionErrorText)
		return
	}

	h.showCustomerTaskAssignmentDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, volunteerID, h.messages.CustomerTaskRejectSuccessText)
}

func (h *MessageHandler) showCustomerTaskDetail(ctx context.Context, chatID, userID int64, taskID string, intro ...string) {
	task, err := h.getTaskByID(ctx, taskID)
	if err != nil || task == nil {
		h.logger.Error("failed to fetch task detail", zap.Error(err), zap.String("task_id", taskID))
		h.renderMenu(ctx, chatID, userID, h.taskFetchErrorText(), h.customerBackKeyboard())
		return
	}

	var builder strings.Builder
	if len(intro) > 0 && strings.TrimSpace(intro[0]) != "" {
		builder.WriteString(strings.TrimSpace(intro[0]))
		builder.WriteString("\n\n")
	}

	title := h.messages.CustomerTaskDetailTitle
	if strings.TrimSpace(title) == "" {
		title = "*%s*"
	}

	name := safeTaskName(task.GetName())
	builder.WriteString(fmt.Sprintf(title, name))
	builder.WriteString("\n\n")
	builder.WriteString(safeTaskDescription(task.GetDescription()))
	builder.WriteString("\n\n")

	assignments := parseTaskAssignments(task)
	keyboard := h.api.Messages.NewKeyboardBuilder()

	if len(assignments) == 0 {
		builder.WriteString(h.customerTasksEmptyText())
		builder.WriteString("\n")
	} else {
		builder.WriteString("🧑‍🤝‍🧑 *Откликнувшиеся:*\n")
		namesCache := make(map[string]string, len(assignments))
		for idx, assignment := range assignments {
			if assignment.UserID == "" {
				continue
			}
			displayName := namesCache[assignment.UserID]
			if displayName == "" {
				displayName = h.lookupUserName(ctx, assignment.UserID)
				namesCache[assignment.UserID] = displayName
			}
			builder.WriteString(fmt.Sprintf("%s — %s\n", displayName, customerStatusLabel(assignment.Status)))

			buttonLabel := truncateLabel(fmt.Sprintf("%d. %s %s", idx+1, displayName, volunteerStatusBadge(assignment.Status)), 45)
			keyboard.AddRow().
				AddCallback(buttonLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s:%s", callbackCustomerTaskAssignment, taskID, assignment.UserID))
		}
		builder.WriteString("\n")
	}

	createLabel := h.messages.CustomerManageCreateTaskButton
	if strings.TrimSpace(createLabel) == "" {
		createLabel = "Создать задачу"
	}
	backLabel := h.messages.CustomerManageBackButton
	if strings.TrimSpace(backLabel) == "" {
		backLabel = "⬅️ Назад"
	}

	keyboard.AddRow().
		AddCallback(createLabel, schemes.POSITIVE, callbackCustomerManageCreateTask)
	keyboard.AddRow().
		AddCallback(backLabel, schemes.DEFAULT, callbackCustomerManageBack)

	h.renderMenu(ctx, chatID, userID, builder.String(), keyboard)
}

func truncateLabel(label string, max int) string {
	if len([]rune(label)) <= max {
		return label
	}
	runes := []rune(label)
	return string(runes[:max-1]) + "…"
}

func (h *MessageHandler) handleCustomerTaskAssignment(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, data string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || data == "" {
		return
	}

	taskID, volunteerID, ok := splitTaskAssignmentData(data)
	if !ok {
		return
	}

	h.showCustomerTaskAssignmentDetail(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, taskID, volunteerID)
}

func (h *MessageHandler) showCustomerTaskAssignmentDetail(ctx context.Context, chatID, userID int64, taskID, volunteerID string, intro ...string) {
	task, err := h.getTaskByID(ctx, taskID)
	if err != nil || task == nil {
		h.logger.Error("failed to fetch task detail", zap.Error(err), zap.String("task_id", taskID))
		h.showCustomerTasksMenu(ctx, chatID, userID, fmt.Sprintf("%d", userID), 0, h.taskFetchErrorText())
		return
	}

	assignments := parseTaskAssignments(task)
	status := assignmentStatusForUser(assignments, volunteerID)
	if status == "" {
		h.showCustomerTaskDetail(ctx, chatID, userID, taskID, h.messages.CustomerTaskDecisionErrorText)
		return
	}

	var builder strings.Builder
	if len(intro) > 0 && strings.TrimSpace(intro[0]) != "" {
		builder.WriteString(strings.TrimSpace(intro[0]))
		builder.WriteString("\n\n")
	}

	title := h.messages.CustomerTaskDetailTitle
	if strings.TrimSpace(title) == "" {
		title = "*%s*"
	}

	taskName := safeTaskName(task.GetName())
	builder.WriteString(fmt.Sprintf(title, taskName))
	builder.WriteString("\n\n")

	displayName := h.lookupUserName(ctx, volunteerID)
	builder.WriteString(fmt.Sprintf("*Волонтёр:* %s\n", displayName))
	builder.WriteString(fmt.Sprintf("*Статус:* %s\n\n", customerStatusLabel(status)))
	builder.WriteString(safeTaskDescription(task.GetDescription()))

	approveLabel := h.messages.CustomerTaskApproveButton
	if strings.TrimSpace(approveLabel) == "" {
		approveLabel = "Подтвердить выполнение"
	}
	rejectLabel := h.messages.CustomerTaskRejectButton
	if strings.TrimSpace(rejectLabel) == "" {
		rejectLabel = "Отклонить"
	}
	keyboard := h.api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback(approveLabel, schemes.POSITIVE, fmt.Sprintf("%s:%s:%s", callbackCustomerTaskApprove, taskID, volunteerID))
	keyboard.AddRow().
		AddCallback(rejectLabel, schemes.NEGATIVE, fmt.Sprintf("%s:%s:%s", callbackCustomerTaskReject, taskID, volunteerID))
	keyboard.AddRow().
		AddCallback(h.messages.CustomerManageTasksButton, schemes.DEFAULT, callbackCustomerManageTasks)
	keyboard.AddRow().
		AddCallback(h.messages.CustomerManageBackButton, schemes.DEFAULT, fmt.Sprintf("%s:%s", callbackCustomerTaskView, taskID))

	h.renderMenu(ctx, chatID, userID, builder.String(), keyboard)
}

func splitTaskAssignmentData(data string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(data), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	taskID := strings.TrimSpace(parts[0])
	volunteerID := strings.TrimSpace(parts[1])
	if taskID == "" || volunteerID == "" {
		return "", "", false
	}
	return taskID, volunteerID, true
}

func parseTaskAssignments(task *taskpb.Task) []taskAssignment {
	if task == nil {
		return nil
	}

	assignments := make(map[string]string)

	for _, meta := range task.GetMeta() {
		if meta == nil {
			continue
		}

		key := strings.TrimSpace(meta.GetKey())
		value := strings.TrimSpace(meta.GetValue())

		if key == "" && value == "" {
			continue
		}

		var arr []taskAssignmentJSON
		if err := json.Unmarshal([]byte(value), &arr); err == nil && len(arr) > 0 {
			for _, item := range arr {
				if strings.TrimSpace(item.UserID) != "" {
					assignments[strings.TrimSpace(item.UserID)] = strings.TrimSpace(item.Status)
				}
			}
			continue
		}

		var single taskAssignmentJSON
		if err := json.Unmarshal([]byte(value), &single); err == nil && strings.TrimSpace(single.UserID) != "" {
			assignments[strings.TrimSpace(single.UserID)] = strings.TrimSpace(single.Status)
			continue
		}

		if id, status := parseAssignmentValue(value); id != "" {
			assignments[id] = status
			continue
		}

		if id, status := parseAssignmentKeyValue(key, value); id != "" {
			assignments[id] = status
		}
	}

	result := make([]taskAssignment, 0, len(assignments))
	for userID, status := range assignments {
		result = append(result, taskAssignment{UserID: userID, Status: status})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Status == result[j].Status {
			return result[i].UserID < result[j].UserID
		}
		return result[i].Status < result[j].Status
	})

	return result
}

func parseAssignmentValue(value string) (string, string) {
	if value == "" {
		return "", ""
	}

	if strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		if len(parts) >= 2 {
			userID := strings.TrimSpace(parts[0])
			status := strings.TrimSpace(strings.Join(parts[1:], ":"))
			if userID != "" {
				return userID, status
			}
		}
	}

	if strings.Contains(value, "=") {
		parts := strings.Split(value, "=")
		if len(parts) >= 2 {
			userID := strings.TrimSpace(parts[0])
			status := strings.TrimSpace(strings.Join(parts[1:], "="))
			if userID != "" {
				return userID, status
			}
		}
	}

	return "", ""
}

func parseAssignmentKeyValue(key, value string) (string, string) {
	if key == "" {
		return "", ""
	}

	lower := strings.ToLower(key)
	if !strings.Contains(lower, "user") {
		return "", ""
	}

	userID := extractDigits(key)
	if userID == "" {
		parts := strings.Split(key, ":")
		if len(parts) > 1 {
			candidate := strings.TrimSpace(parts[len(parts)-1])
			if candidate != "" && !strings.Contains(strings.ToLower(candidate), "status") && !strings.Contains(strings.ToLower(candidate), "user") {
				userID = candidate
			}
		}
	}

	if userID == "" {
		return "", ""
	}

	status := value
	if status == "" {
		status = statusFromKey(lower)
	}

	return userID, status
}

func extractDigits(input string) string {
	var builder strings.Builder
	for _, r := range input {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func statusFromKey(key string) string {
	switch {
	case strings.Contains(key, "approve"):
		return "approved"
	case strings.Contains(key, "reject") || strings.Contains(key, "decline"):
		return "rejected"
	case strings.Contains(key, "confirm") || strings.Contains(key, "complete") || strings.Contains(key, "done"):
		return "confirmed"
	case strings.Contains(key, "pending") || strings.Contains(key, "wait"):
		return "pending"
	default:
		return ""
	}
}

func assignmentStatusForUser(assignments []taskAssignment, userID string) string {
	for _, assignment := range assignments {
		if assignment.UserID == userID {
			return assignment.Status
		}
	}
	return ""
}

func normalizeStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func volunteerStatusBadge(status string) string {
	switch normalizeStatus(status) {
	case "pending", "waiting", "new":
		return "⏳"
	case "approved", "accept", "accepted", "in_progress":
		return "✅"
	case "rejected", "declined", "cancelled":
		return "❌"
	case "confirmed", "completed", "done":
		return "✨"
	default:
		return ""
	}
}

func volunteerStatusLabel(status string) string {
	switch normalizeStatus(status) {
	case "":
		return ""
	case "pending", "waiting", "new":
		return "⏳ ждёт подтверждения"
	case "approved", "accept", "accepted", "in_progress":
		return "✅ подтверждено заказчиком"
	case "rejected", "declined", "cancelled":
		return "❌ отклонено"
	case "confirmed", "completed", "done":
		return "✨ выполнено"
	default:
		return strings.Title(normalizeStatus(status))
	}
}

func customerStatusLabel(status string) string {
	switch normalizeStatus(status) {
	case "":
		return "—"
	case "pending", "waiting", "new":
		return "⏳ ждёт подтверждения"
	case "approved", "accept", "accepted", "in_progress":
		return "✅ ждёт подтверждения выполнения"
	case "rejected", "declined", "cancelled":
		return "❌ отклонено"
	case "confirmed", "completed", "done":
		return "✨ выполнено"
	default:
		return strings.Title(normalizeStatus(status))
	}
}

func isStatusRejected(status string) bool {
	switch normalizeStatus(status) {
	case "rejected", "declined", "cancelled":
		return true
	default:
		return false
	}
}

func allowVolunteerJoin(status string) bool {
	switch normalizeStatus(status) {
	case "", "rejected", "declined", "cancelled":
		return true
	default:
		return false
	}
}

func allowVolunteerLeave(status string) bool {
	switch normalizeStatus(status) {
	case "", "rejected", "declined", "cancelled", "confirmed", "completed", "done":
		return false
	default:
		return true
	}
}

func allowVolunteerConfirm(status string) bool {
	switch normalizeStatus(status) {
	case "approved", "accept", "accepted", "in_progress":
		return true
	default:
		return false
	}
}

func safeTaskName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Без названия"
	}
	return name
}

func safeTaskDescription(desc string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return "Описание отсутствует"
	}
	return desc
}

func (h *MessageHandler) getTaskByID(ctx context.Context, id string) (*taskpb.Task, error) {
	if h.task == nil {
		return nil, fmt.Errorf("task service client is not configured")
	}

	resp, err := h.task.GetTaskByID(ctx, &taskpb.GetTaskByIDRequest{Id: id})
	if err != nil {
		return nil, err
	}

	if resp.GetError() != nil {
		if resp.GetError().GetCode() == taskpb.ErrorCode_ERROR_CODE_NOT_FOUND {
			return nil, nil
		}
		return nil, fmt.Errorf("task service error: %s", resp.GetError().GetMessage())
	}

	return resp.GetTask(), nil
}
