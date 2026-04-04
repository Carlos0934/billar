package app

import "context"

type SessionService interface {
	Status(ctx context.Context) (SessionStatusDTO, error)
}

type SessionStatusDTO struct {
	Status        string `json:"status" toon:"status"`
	Email         string `json:"email,omitempty" toon:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty" toon:"email_verified,omitempty"`
	Subject       string `json:"subject,omitempty" toon:"subject,omitempty"`
	Issuer        string `json:"issuer,omitempty" toon:"issuer,omitempty"`
}
