package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/users"
)

var _ users.EmailVerificationRepository = (*emailVerificationsRepositoryMock)(nil)

type emailVerificationsRepositoryMock struct {
	mu                   sync.Mutex
	verificationsByToken map[string]users.EmailVerification
}

func NewEmailVerificationRepository(vs []users.EmailVerification) users.EmailVerificationRepository {
	repo := &emailVerificationsRepositoryMock{
		verificationsByToken: make(map[string]users.EmailVerification, len(vs)),
	}

	for _, v := range vs {
		repo.verificationsByToken[v.Token] = v
	}

	return repo
}

func (vrm *emailVerificationsRepositoryMock) Save(ctx context.Context, verification users.EmailVerification) (string, error) {
	vrm.mu.Lock()
	defer vrm.mu.Unlock()

	if _, ok := vrm.verificationsByToken[verification.Token]; ok {
		return "", dbutil.ErrConflict
	}

	vrm.verificationsByToken[verification.Token] = verification
	return verification.Token, nil
}

func (vrm *emailVerificationsRepositoryMock) RetrieveByToken(ctx context.Context, confirmationToken string) (users.EmailVerification, error) {
	vrm.mu.Lock()
	defer vrm.mu.Unlock()

	v, ok := vrm.verificationsByToken[confirmationToken]
	if !ok {
		return users.EmailVerification{}, dbutil.ErrNotFound
	}

	return v, nil
}

func (vrm *emailVerificationsRepositoryMock) Remove(ctx context.Context, confirmationToken string) error {
	vrm.mu.Lock()
	defer vrm.mu.Unlock()

	if _, ok := vrm.verificationsByToken[confirmationToken]; !ok {
		return dbutil.ErrNotFound
	}

	delete(vrm.verificationsByToken, confirmationToken)

	return nil
}
