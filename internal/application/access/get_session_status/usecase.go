package getsessionstatus

import (
	"context"

	"github.com/Carlos0934/billar/internal/application/access/access_dto"
	"github.com/Carlos0934/billar/internal/application/access/ports"
)

type Result = accessdto.SessionStatus

type UseCase struct {
	store ports.CurrentUnlockedSessionStore
}

func NewUseCase(store ports.CurrentUnlockedSessionStore) *UseCase {
	return &UseCase{store: store}
}

func (useCase *UseCase) Execute(ctx context.Context) (Result, error) {
	current, err := useCase.store.GetCurrent(ctx)
	if err != nil {
		return accessdto.LockedSessionStatus(), err
	}
	if current == nil {
		return accessdto.LockedSessionStatus(), nil
	}

	return accessdto.UnlockedSessionStatus(current), nil
}
