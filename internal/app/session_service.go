package app

import "context"

type SessionService interface {
	StartLogin(ctx context.Context) (LoginIntentDTO, error)
	Status(ctx context.Context) (SessionStatusDTO, error)
	Logout(ctx context.Context) (LogoutDTO, error)
}

type LoginIntentDTO struct {
	LoginURL string `json:"login_url" toon:"login_url"`
}

type SessionStatusDTO struct {
	Status        string `json:"status" toon:"status"`
	Email         string `json:"email,omitempty" toon:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty" toon:"email_verified,omitempty"`
	Subject       string `json:"subject,omitempty" toon:"subject,omitempty"`
	Issuer        string `json:"issuer,omitempty" toon:"issuer,omitempty"`
}

type LogoutDTO struct {
	Message string `json:"message" toon:"message"`
}
