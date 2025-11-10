package handlers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	taskpb "DobrikaDev/max-bot/internal/generated/taskpb"

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

	h.showCustomerTasksMenu(ctx, chatID, userID, strings.TrimSpace(customer.GetMaxId()))
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
	h.showCustomerTasksMenu(ctx, session.ChatID, session.UserID, session.CustomerID, success)
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

func (h *MessageHandler) showCustomerTasksMenu(ctx context.Context, chatID, userID int64, customerID string, intro ...string) {
	text, keyboard := h.buildCustomerTasksView(ctx, customerID, intro...)
	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) buildCustomerTasksView(ctx context.Context, customerID string, intro ...string) (string, *maxbot.Keyboard) {
	var builder strings.Builder
	if len(intro) > 0 && strings.TrimSpace(intro[0]) != "" {
		builder.WriteString(strings.TrimSpace(intro[0]))
		builder.WriteString("\n\n")
	}

	if h.task == nil {
		builder.WriteString(h.taskServiceUnavailableText())
		return builder.String(), h.customerBackKeyboard()
	}

	resp, err := h.task.GetTasks(ctx, &taskpb.GetTasksRequest{CustomerId: customerID, Limit: 10, Offset: 0})
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

	tasks := resp.GetTasks()
	keyboard := h.api.Messages.NewKeyboardBuilder()

	if len(tasks) == 0 {
		builder.WriteString(h.customerTasksEmptyText())
	} else {
		title := strings.TrimSpace(h.messages.CustomerTasksListText)
		if title != "" {
			builder.WriteString(title)
			builder.WriteString("\n\n")
		}
		itemTemplate := h.customerTaskItemTemplate()
		for idx, task := range tasks {
			name := strings.TrimSpace(task.GetName())
			if name == "" {
				name = "Без названия"
			}
			description := strings.TrimSpace(task.GetDescription())
			if description == "" {
				description = "Описание отсутствует"
			}
			builder.WriteString(fmt.Sprintf(itemTemplate, name, description))
			builder.WriteString("\n")

			label := truncateLabel(fmt.Sprintf("%d. %s", idx+1, name), 40)
			keyboard.AddRow().
				AddCallback(label, schemes.DEFAULT, fmt.Sprintf("%s:%s", callbackCustomerTaskView, task.GetId()))
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

func (h *MessageHandler) showVolunteerTasksList(ctx context.Context, chatID, userID int64, intro string) {
	text, keyboard := h.buildVolunteerTasksView(ctx, intro)
	h.renderMenu(ctx, chatID, userID, text, keyboard)
}

func (h *MessageHandler) buildVolunteerTasksView(ctx context.Context, intro string) (string, *maxbot.Keyboard) {
	var builder strings.Builder
	if strings.TrimSpace(intro) != "" {
		builder.WriteString(strings.TrimSpace(intro))
		builder.WriteString("\n\n")
	}

	if h.task == nil {
		builder.WriteString(h.volunteerTasksUnavailableText())
		return builder.String(), h.volunteerBackKeyboard()
	}

	resp, err := h.task.GetTasks(ctx, &taskpb.GetTasksRequest{Limit: 10, Offset: 0})
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

	tasks := resp.GetTasks()
	if len(tasks) == 0 {
		builder.WriteString(h.volunteerTasksEmptyText())
	} else {
		itemTemplate := h.volunteerTaskItemTemplate()
		keyboard := h.api.Messages.NewKeyboardBuilder()
		for idx, task := range tasks {
			name := strings.TrimSpace(task.GetName())
			if name == "" {
				name = "Без названия"
			}
			description := strings.TrimSpace(task.GetDescription())
			if description == "" {
				description = "Описание отсутствует"
			}
			builder.WriteString(fmt.Sprintf(itemTemplate, name, description))
			builder.WriteString("\n")

			label := truncateLabel(fmt.Sprintf("%d. %s", idx+1, name), 40)
			keyboard.AddRow().
				AddCallback(label, schemes.DEFAULT, fmt.Sprintf("%s:%s", callbackVolunteerTaskView, task.GetId()))
		}
		keyboard.AddRow().
			AddCallback(h.messages.VolunteerMenuBackButton, schemes.DEFAULT, callbackVolunteerBack)
		keyboard.AddRow().
			AddCallback(h.messages.VolunteerMenuMainButton, schemes.DEFAULT, callbackProfileBack)
		return builder.String(), keyboard
	}

	return builder.String(), h.volunteerBackKeyboard()
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

func (h *MessageHandler) volunteerTaskItemTemplate() string {
	if text := strings.TrimSpace(h.messages.VolunteerTaskItemTemplate); text != "" {
		return text
	}
	return "• *%s*\n%s"
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
		h.showVolunteerTasksList(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, h.volunteerTasksUnavailableText())
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
		h.showVolunteerTasksList(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, h.volunteerTasksUnavailableText())
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
		h.showVolunteerTasksList(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, h.volunteerTasksUnavailableText())
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
		h.showVolunteerTasksList(ctx, chatID, userID, h.volunteerTasksUnavailableText())
		return
	}

	task, err := h.getTaskByID(ctx, taskID)
	if err != nil || task == nil {
		h.logger.Error("failed to fetch task detail", zap.Error(err), zap.String("task_id", taskID))
		h.showVolunteerTasksList(ctx, chatID, userID, h.volunteerTasksErrorText())
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

	name := strings.TrimSpace(task.GetName())
	if name == "" {
		name = "Без названия"
	}
	builder.WriteString(fmt.Sprintf(title, name))
	builder.WriteString("\n\n")

	description := strings.TrimSpace(task.GetDescription())
	if description == "" {
		description = "Описание отсутствует"
	}
	builder.WriteString(description)

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

	keyboard.AddRow().
		AddCallback(joinLabel, schemes.POSITIVE, fmt.Sprintf("%s:%s", callbackVolunteerTaskJoin, taskID))
	keyboard.AddRow().
		AddCallback(leaveLabel, schemes.DEFAULT, fmt.Sprintf("%s:%s", callbackVolunteerTaskLeave, taskID)).
		AddCallback(confirmLabel, schemes.POSITIVE, fmt.Sprintf("%s:%s", callbackVolunteerTaskConfirm, taskID))
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

func (h *MessageHandler) handleCustomerTaskApprove(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, taskID string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || taskID == "" {
		return
	}

	if h.task == nil {
		h.renderMenu(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, h.taskServiceUnavailableText(), h.customerBackKeyboard())
		return
	}

	userID := fmt.Sprintf("%d", callbackQuery.Callback.User.UserId)
	chatID := callbackQuery.Message.Recipient.ChatId

	resp, err := h.task.ApproveTask(ctx, &taskpb.ApproveTaskRequest{UserId: userID, TaskId: taskID})
	if err != nil || (resp != nil && resp.GetError() != nil) {
		h.showCustomerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.CustomerTaskDecisionErrorText)
		return
	}

	h.showCustomerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.CustomerTaskApproveSuccessText)
}

func (h *MessageHandler) handleCustomerTaskReject(ctx context.Context, callbackQuery *schemes.MessageCallbackUpdate, taskID string) {
	h.answerCallback(ctx, callbackQuery.Callback.CallbackID)

	if callbackQuery.Message == nil || taskID == "" {
		return
	}

	if h.task == nil {
		h.renderMenu(ctx, callbackQuery.Message.Recipient.ChatId, callbackQuery.Callback.User.UserId, h.taskServiceUnavailableText(), h.customerBackKeyboard())
		return
	}

	userID := fmt.Sprintf("%d", callbackQuery.Callback.User.UserId)
	chatID := callbackQuery.Message.Recipient.ChatId

	resp, err := h.task.RejectTask(ctx, &taskpb.RejectTaskRequest{UserId: userID, TaskId: taskID})
	if err != nil || (resp != nil && resp.GetError() != nil) {
		h.showCustomerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.CustomerTaskDecisionErrorText)
		return
	}

	h.showCustomerTaskDetail(ctx, chatID, callbackQuery.Callback.User.UserId, taskID, h.messages.CustomerTaskRejectSuccessText)
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

	name := strings.TrimSpace(task.GetName())
	if name == "" {
		name = "Без названия"
	}
	builder.WriteString(fmt.Sprintf(title, name))
	builder.WriteString("\n\n")

	description := strings.TrimSpace(task.GetDescription())
	if description == "" {
		description = "Описание отсутствует"
	}
	builder.WriteString(description)

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
		AddCallback(approveLabel, schemes.POSITIVE, fmt.Sprintf("%s:%s", callbackCustomerTaskApprove, taskID))
	keyboard.AddRow().
		AddCallback(rejectLabel, schemes.NEGATIVE, fmt.Sprintf("%s:%s", callbackCustomerTaskReject, taskID))
	keyboard.AddRow().
		AddCallback(h.messages.CustomerManageTasksButton, schemes.DEFAULT, callbackCustomerManageTasks)

	h.renderMenu(ctx, chatID, userID, builder.String(), keyboard)
}

func truncateLabel(label string, max int) string {
	if len([]rune(label)) <= max {
		return label
	}
	runes := []rune(label)
	return string(runes[:max-1]) + "…"
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
