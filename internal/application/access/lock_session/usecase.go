package locksession

import (
	"context"

	"github.com/Carlos0934/billar/internal/application/access/access_dto"
	"github.com/Carlos0934/billar/internal/application/access/ports"
)

type Result = accessdto.SessionStatus

type UseCase struct {
	store ports.CurrentUnlockedSessionStore
	clock ports.Clock
}

func NewUseCase(store ports.CurrentUnlockedSessionStore, clock ports.Clock) *UseCase {
	return &UseCase{store: store, clock: clock}
}

func (useCase *UseCase) Execute(ctx context.Context) (Result, error) {
	current, err := useCase.store.GetCurrent(ctx)
	if err != nil {
		return accessdto.LockedSessionStatus(), err
	}
	if current == nil {
		return accessdto.LockedSessionStatus(), nil
	}

	if err := current.Lock(useCase.clock.Now()); err != nil {
		return accessdto.LockedSessionStatus(), err
	}
	if err := useCase.store.DeleteCurrent(ctx); err != nil {
		return accessdto.LockedSessionStatus(), err
	}

	return accessdto.LockedSessionStatus(), nil
}
