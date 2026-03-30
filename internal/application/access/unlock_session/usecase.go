package unlocksession

import (
	"context"

	"github.com/Carlos0934/billar/internal/application/access/access_dto"
	"github.com/Carlos0934/billar/internal/application/access/ports"
	"github.com/Carlos0934/billar/internal/domain/access/session"
)

type Result = accessdto.SessionStatus

type Command struct {
	Secret string
}

type UseCase struct {
	store       ports.CurrentUnlockedSessionStore
	verifier    ports.UnlockSecretVerifier
	clock       ports.Clock
	idGenerator ports.SessionIDGenerator
}

func NewUseCase(
	store ports.CurrentUnlockedSessionStore,
	verifier ports.UnlockSecretVerifier,
	clock ports.Clock,
	idGenerator ports.SessionIDGenerator,
) *UseCase {
	return &UseCase{
		store:       store,
		verifier:    verifier,
		clock:       clock,
		idGenerator: idGenerator,
	}
}

func (useCase *UseCase) Execute(ctx context.Context, command Command) (Result, error) {
	current, err := useCase.store.GetCurrent(ctx)
	if err != nil {
		return accessdto.LockedSessionStatus(), err
	}
	if current != nil {
		return accessdto.UnlockedSessionStatus(current), nil
	}

	if err := useCase.verifier.Verify(ctx, command.Secret); err != nil {
		return accessdto.LockedSessionStatus(), err
	}

	current, err = session.New(session.CreateParams{
		ID:         useCase.idGenerator.New(),
		UnlockedAt: useCase.clock.Now(),
	})
	if err != nil {
		return accessdto.LockedSessionStatus(), err
	}

	if err := useCase.store.SaveCurrent(ctx, current); err != nil {
		return accessdto.LockedSessionStatus(), err
	}

	return accessdto.UnlockedSessionStatus(current), nil
}
