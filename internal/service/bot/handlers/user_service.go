package handlers

import (
	"context"
	"fmt"

	userpb "DobrikaDev/max-bot/internal/generated/userpb"

	"go.uber.org/zap"
)

func (h *MessageHandler) sendRegistrationToUserService(ctx context.Context, session *registrationSession) error {
	if h.user == nil {
		return fmt.Errorf("user service client is not configured")
	}

	req := &userpb.CreateUserRequest{
		User: &userpb.User{
			MaxId:       session.MaxUserID,
			Name:        session.UserName,
			Geolocation: session.geolocationAsString(),
			Age:         session.Age,
			Sex:         session.Sex,
			About:       session.About,
			Role:        userpb.Role_ROLE_USER,
			Status:      userpb.Status_STATUS_ACTIVE,
		},
	}

	resp, err := h.user.CreateUser(ctx, req)
	if err != nil {
		return fmt.Errorf("create user request failed: %w", err)
	}

	if resp.GetError() != nil {
		return fmt.Errorf("user service responded with error: %s", resp.GetError().GetMessage())
	}

	h.logger.Info("user registration stored", zap.String("max_id", session.MaxUserID))
	return nil
}
