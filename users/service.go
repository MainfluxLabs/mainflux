// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"regexp"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

const (
	EnabledStatusKey  = "enabled"
	DisabledStatusKey = "disabled"
	AllStatusKey      = "all"
	rootAdminRole     = "root"
)

var (
	// ErrMissingResetToken indicates malformed or missing reset token
	// for reseting password.
	ErrMissingResetToken = errors.New("missing reset token")

	// ErrRecoveryToken indicates error in generating password recovery token.
	ErrRecoveryToken = errors.New("failed to generate password recovery token")

	// ErrGetToken indicates error in getting signed token.
	ErrGetToken = errors.New("failed to fetch signed token")

	// ErrPasswordFormat indicates weak password.
	ErrPasswordFormat = errors.New("password does not meet the requirements")

	// ErrAlreadyEnabledUser indicates the user is already enabled.
	ErrAlreadyEnabledUser = errors.New("the user is already enabled")

	// ErrAlreadyDisabledUser indicates the user is already disabled.
	ErrAlreadyDisabledUser = errors.New("the user is already disabled")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Register creates new user account. In case of the failed registration, a
	// non-nil error value is returned. The user registration is only allowed
	// for admin.
	SelfRegister(ctx context.Context, user User) (string, error)

	// Register creates new user account. In case of the failed registration, a
	// non-nil error value is returned. The user registration is only allowed
	// for admin.
	Register(ctx context.Context, token string, user User) (string, error)

	// RegisterAdmin creates new root admin account. In case of the failed registration, a
	// non-nil error value is returned. The user registration is only allowed
	// for root admin.
	RegisterAdmin(ctx context.Context, user User) error

	// Login authenticates the user given its credentials. Successful
	// authentication generates new access token. Failed invocations are
	// identified by the non-nil error values in the response.
	Login(ctx context.Context, user User) (string, error)

	// ViewUser retrieves user info for a given user ID and an authorized token.
	ViewUser(ctx context.Context, token, id string) (User, error)

	// ViewProfile retrieves user info for a given token.
	ViewProfile(ctx context.Context, token string) (User, error)

	// ListUsers retrieves users list for a valid admin token.
	ListUsers(ctx context.Context, token string, pm PageMetadata) (UserPage, error)

	// ListUsersByIDs retrieves users list for the given IDs.
	ListUsersByIDs(ctx context.Context, ids []string) (UserPage, error)

	// ListUsersByEmails retrieves users list for the given emails.
	ListUsersByEmails(ctx context.Context, emails []string) ([]User, error)

	// UpdateUser updates the user metadata.
	UpdateUser(ctx context.Context, token string, user User) error

	// GenerateResetToken email where mail will be sent.
	// host is used for generating reset link.
	GenerateResetToken(ctx context.Context, email, host string) error

	// ChangePassword change users password for authenticated user.
	ChangePassword(ctx context.Context, authToken, password, oldPassword string) error

	// ResetPassword change users password in reset flow.
	// token can be authentication token or password reset token.
	ResetPassword(ctx context.Context, resetToken, password string) error

	// SendPasswordReset sends reset password link to email.
	SendPasswordReset(ctx context.Context, host, email, token string) error

	// EnableUser logically enableds the user identified with the provided ID
	EnableUser(ctx context.Context, token, id string) error

	// DisableUser logically disables the user identified with the provided ID
	DisableUser(ctx context.Context, token, id string) error

	// Backup returns admin and all users. Only accessible by admin.
	Backup(ctx context.Context, token string) (User, []User, error)

	// Restore restores users from backup. Only accessible by admin.
	Restore(ctx context.Context, token string, admin User, users []User) error
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Email    string
	Status   string
	Metadata Metadata
}

// UserPage contains a page of users.
type UserPage struct {
	PageMetadata
	Users []User
}

var _ Service = (*usersService)(nil)

type usersService struct {
	users      UserRepository
	hasher     Hasher
	email      Emailer
	auth       protomfx.AuthServiceClient
	idProvider uuid.IDProvider
	passRegex  *regexp.Regexp
}

// New instantiates the users service implementation
func New(users UserRepository, hasher Hasher, auth protomfx.AuthServiceClient, e Emailer, idp uuid.IDProvider, passRegex *regexp.Regexp) Service {
	return &usersService{
		users:      users,
		hasher:     hasher,
		auth:       auth,
		email:      e,
		idProvider: idp,
		passRegex:  passRegex,
	}
}

func (svc usersService) SelfRegister(ctx context.Context, user User) (string, error) {
	if !svc.passRegex.MatchString(user.Password) {
		return "", ErrPasswordFormat
	}

	uid, err := svc.idProvider.ID()
	if err != nil {
		return "", err
	}
	user.ID = uid

	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}
	user.Password = hash

	user.Status = EnabledStatusKey

	uid, err = svc.users.Save(ctx, user)
	if err != nil {
		return "", err
	}
	return uid, nil
}

func (svc usersService) RegisterAdmin(ctx context.Context, user User) error {
	if u, err := svc.users.RetrieveByEmail(context.Background(), user.Email); err == nil {
		role, err := svc.auth.RetrieveRole(ctx, &protomfx.RetrieveRoleReq{Id: u.ID})
		if err != nil {
			return err
		}

		req := protomfx.AssignRoleReq{
			Id:   u.ID,
			Role: auth.RoleRootAdmin,
		}

		switch role.Role {
		case auth.RoleRootAdmin:
			return nil
		default:
			if _, err := svc.auth.AssignRole(ctx, &req); err != nil {
				return err
			}
		}

		return nil
	}

	if !svc.passRegex.MatchString(user.Password) {
		return ErrPasswordFormat
	}

	uid, err := svc.idProvider.ID()
	if err != nil {
		return err
	}
	user.ID = uid

	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, err)
	}
	user.Password = hash

	user.Status = EnabledStatusKey

	if _, err := svc.users.Save(ctx, user); err != nil {
		return err
	}

	req := protomfx.AssignRoleReq{
		Id:   user.ID,
		Role: auth.RoleRootAdmin,
	}

	if _, err := svc.auth.AssignRole(ctx, &req); err != nil {
		return err
	}

	return nil
}

func (svc usersService) Register(ctx context.Context, token string, user User) (string, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		return "", err
	}

	if !svc.passRegex.MatchString(user.Password) {
		return "", ErrPasswordFormat
	}

	uid, err := svc.idProvider.ID()
	if err != nil {
		return "", err
	}

	user.ID = uid

	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}
	user.Password = hash
	if user.Status == "" {
		user.Status = EnabledStatusKey
	}

	if user.Status != AllStatusKey &&
		user.Status != EnabledStatusKey &&
		user.Status != DisabledStatusKey {
		return "", apiutil.ErrInvalidStatus
	}

	uid, err = svc.users.Save(ctx, user)
	if err != nil {
		return "", err
	}
	return uid, nil
}

func (svc usersService) Login(ctx context.Context, user User) (string, error) {
	dbUser, err := svc.users.RetrieveByEmail(ctx, user.Email)
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthentication, err)
	}
	if err := svc.hasher.Compare(user.Password, dbUser.Password); err != nil {
		return "", errors.Wrap(errors.ErrAuthentication, err)
	}
	return svc.issue(ctx, dbUser.ID, dbUser.Email, auth.LoginKey)
}

func (svc usersService) ViewUser(ctx context.Context, token, id string) (User, error) {
	if _, err := svc.identify(ctx, token); err != nil {
		return User{}, err
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return User{}, errors.Wrap(errors.ErrNotFound, err)
	}

	return User{
		ID:       id,
		Email:    dbUser.Email,
		Password: "",
		Metadata: dbUser.Metadata,
		Status:   dbUser.Status,
	}, nil
}

func (svc usersService) ViewProfile(ctx context.Context, token string) (User, error) {
	ir, err := svc.identify(ctx, token)
	if err != nil {
		return User{}, err
	}

	dbUser, err := svc.users.RetrieveByID(ctx, ir.id)
	if err != nil {
		return User{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	u := User{
		ID:       dbUser.ID,
		Email:    ir.email,
		Metadata: dbUser.Metadata,
	}

	if err := svc.isAdmin(ctx, token); err != nil {
		return u, nil
	}

	u.Role = rootAdminRole

	return u, nil
}

func (svc usersService) ListUsers(ctx context.Context, token string, pm PageMetadata) (UserPage, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		return UserPage{}, err
	}

	return svc.users.RetrieveByIDs(ctx, nil, pm)
}

func (svc usersService) ListUsersByIDs(ctx context.Context, ids []string) (UserPage, error) {
	pm := PageMetadata{Status: EnabledStatusKey}
	return svc.users.RetrieveByIDs(ctx, ids, pm)
}

func (svc usersService) ListUsersByEmails(ctx context.Context, emails []string) ([]User, error) {
	var users []User
	for _, email := range emails {
		u, err := svc.users.RetrieveByEmail(ctx, email)
		if err != nil {
			return []User{}, err
		}
		users = append(users, u)
	}

	return users, nil
}

func (svc usersService) Backup(ctx context.Context, token string) (User, []User, error) {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return User{}, []User{}, err
	}

	if err := svc.isAdmin(ctx, token); err != nil {
		return User{}, []User{}, err
	}

	users, err := svc.users.RetrieveAll(ctx)
	if err != nil {
		return User{}, []User{}, err
	}

	var admin User
	for i, u := range users {
		if u.Email == user.email {
			admin = u
			users = append(users[:i], users[i+1:]...)
			break
		}
	}

	return admin, users, nil
}

func (svc usersService) Restore(ctx context.Context, token string, admin User, users []User) error {
	if err := svc.isAdmin(ctx, token); err != nil {
		return err
	}

	if err := svc.users.UpdateUser(ctx, admin); err != nil {
		return err
	}

	req := protomfx.AssignRoleReq{
		Id:   admin.ID,
		Role: auth.RoleRootAdmin,
	}

	if _, err := svc.auth.AssignRole(ctx, &req); err != nil {
		return err
	}

	for _, user := range users {
		if _, err := svc.users.Save(ctx, user); err != nil {
			return err
		}
	}

	return nil
}

func (svc usersService) UpdateUser(ctx context.Context, token string, u User) error {
	idn, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	user := User{
		Email:    idn.email,
		Metadata: u.Metadata,
	}
	return svc.users.UpdateUser(ctx, user)
}

func (svc usersService) GenerateResetToken(ctx context.Context, email, host string) error {
	user, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil || user.Email == "" {
		return errors.ErrNotFound
	}
	t, err := svc.issue(ctx, user.ID, user.Email, auth.RecoveryKey)
	if err != nil {
		return errors.Wrap(ErrRecoveryToken, err)
	}
	return svc.SendPasswordReset(ctx, host, email, t)
}

func (svc usersService) ResetPassword(ctx context.Context, resetToken, password string) error {
	ir, err := svc.identify(ctx, resetToken)
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}
	u, err := svc.users.RetrieveByID(ctx, ir.id)
	if err != nil {
		return err
	}
	if u.Email == "" {
		return errors.ErrNotFound
	}
	if !svc.passRegex.MatchString(password) {
		return ErrPasswordFormat
	}
	password, err = svc.hasher.Hash(password)
	if err != nil {
		return err
	}
	return svc.users.UpdatePassword(ctx, ir.email, password)
}

func (svc usersService) ChangePassword(ctx context.Context, authToken, password, oldPassword string) error {
	ir, err := svc.identify(ctx, authToken)
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}
	if !svc.passRegex.MatchString(password) {
		return ErrPasswordFormat
	}
	u := User{
		Email:    ir.email,
		ID:       ir.id,
		Password: oldPassword,
	}
	if _, err := svc.Login(ctx, u); err != nil {
		return errors.ErrAuthentication
	}
	u, err = svc.users.RetrieveByID(ctx, ir.id)
	if err != nil || u.Email == "" {
		return errors.ErrNotFound
	}

	password, err = svc.hasher.Hash(password)
	if err != nil {
		return err
	}
	return svc.users.UpdatePassword(ctx, ir.email, password)
}

func (svc usersService) SendPasswordReset(_ context.Context, host, email, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, host, token)
}

func (svc usersService) EnableUser(ctx context.Context, token, id string) error {
	if err := svc.changeStatus(ctx, token, id, EnabledStatusKey); err != nil {
		return err
	}
	return nil
}

func (svc usersService) DisableUser(ctx context.Context, token, id string) error {
	if err := svc.changeStatus(ctx, token, id, DisabledStatusKey); err != nil {
		return err
	}
	return nil
}

func (svc usersService) changeStatus(ctx context.Context, token, id, status string) error {
	if _, err := svc.identify(ctx, token); err != nil {
		return err
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(errors.ErrNotFound, err)
	}
	if dbUser.Status == status {
		if status == DisabledStatusKey {
			return ErrAlreadyDisabledUser
		}
		return ErrAlreadyEnabledUser
	}

	return svc.users.ChangeStatus(ctx, id, status)
}

// Auth helpers
func (svc usersService) issue(ctx context.Context, id, email string, keyType uint32) (string, error) {
	key, err := svc.auth.Issue(ctx, &protomfx.IssueReq{Id: id, Email: email, Type: keyType})
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}
	return key.GetValue(), nil
}

type userIdentity struct {
	id    string
	email string
}

func (svc usersService) identify(ctx context.Context, token string) (userIdentity, error) {
	identity, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return userIdentity{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	return userIdentity{identity.Id, identity.Email}, nil
}

func (svc usersService) isAdmin(ctx context.Context, token string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Subject: auth.RootSubject,
	}

	if _, err := svc.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}
