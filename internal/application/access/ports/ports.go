package ports

import (
	"context"
	"time"

	"github.com/Carlos0934/billar/internal/domain/access/session"
)

type CurrentUnlockedSessionStore interface {
	GetCurrent(ctx context.Context) (*session.Session, error)
	SaveCurrent(ctx context.Context, current *session.Session) error
	DeleteCurrent(ctx context.Context) error
}

type UnlockSecretVerifier interface {
	Verify(ctx context.Context, secret string) error
}

type Clock interface {
	Now() time.Time
}

type SessionIDGenerator interface {
	New() session.SessionID
}
