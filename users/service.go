// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"golang.org/x/oauth2"
)

const (
	EnabledStatusKey  = "enabled"
	DisabledStatusKey = "disabled"
	AllStatusKey      = "all"
	rootAdminRole     = "root"
	GoogleProvider    = "google"
	GitHubProvider    = "github"
)

var (
	// ErrRecoveryToken indicates error in generating password recovery token.
	ErrRecoveryToken = errors.New("failed to generate password recovery token")

	// ErrPasswordFormat indicates weak password.
	ErrPasswordFormat = errors.New("password does not meet the requirements")

	// ErrAlreadyEnabledUser indicates the user is already enabled.
	ErrAlreadyEnabledUser = errors.New("the user is already enabled")

	// ErrAlreadyDisabledUser indicates the user is already disabled.
	ErrAlreadyDisabledUser = errors.New("the user is already disabled")

	// ErrEmailVerificationExpired indicates that the e-mail verification token has expired.
	ErrEmailVerificationExpired = errors.New("e-mail verification token expired")

	// ErrSelfRegisterDisabled indicates that self-registration is disabled in the service config.
	ErrSelfRegisterDisabled = errors.New("self register disabled")
)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// SelfRegister carries out the first stage of own account registration: it
	// creates a pending e-mail verification entity and sends the user an e-mail
	// with a URL containing a token used to verify the e-mail address and complete
	// registration.
	SelfRegister(ctx context.Context, user User, redirectPath string) (string, error)

	// VerifyEmail completes the self-registration process by matching the provided
	// email verification token against the database. If the token is valid and not expired, the e-mail
	// is considered verified a new User is fully registered.
	// Returns the ID of the newly-registered User upon success.
	VerifyEmail(ctx context.Context, confirmationToken string) (string, error)

	// RegisterByInvite performs user registration based on a platform invite.
	// inviteID must correspond to a valid, pending and non-expired platform invite, and the user's supplied
	// e-mail address must match the e-mail address of that platform invite. Upon success, marks the associated
	// invite's state as 'accepted'. Returns the ID of the newly registered user.
	RegisterByInvite(ctx context.Context, user User, inviteID, orgInviteRedirectPath string) (string, error)

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

	// OAuthLogin returns the URL to initiate OAuth login.
	OAuthLogin(provider string) (state, verifier, redirectURL string, err error)

	// OAuthCallback exchanges the OAuth code for user info and logs in/creates the user.
	OAuthCallback(ctx context.Context, provider, code, verifier string) (string, error)

	// ViewUser retrieves user info for a given user ID and an authorized token.
	ViewUser(ctx context.Context, token, id string) (User, error)

	// ViewProfile retrieves user info for a given token.
	ViewProfile(ctx context.Context, token string) (User, error)

	// ListUsers retrieves users list for a valid admin token.
	ListUsers(ctx context.Context, token string, pm PageMetadata) (UserPage, error)

	// ListUsersByIDs retrieves users list for the given IDs.
	ListUsersByIDs(ctx context.Context, ids []string, pm PageMetadata) (UserPage, error)

	// ListUsersByEmails retrieves users list for the given emails.
	ListUsersByEmails(ctx context.Context, emails []string) ([]User, error)

	// UpdateUser updates the user metadata.
	UpdateUser(ctx context.Context, token string, user User) error

	// GenerateResetToken email where mail will be sent.
	GenerateResetToken(ctx context.Context, email, redirectPath string) error

	// ChangePassword change users password for authenticated user.
	ChangePassword(ctx context.Context, token, email, password, oldPassword string) error

	// ResetPassword change users password in reset flow.
	// token can be authentication token or password reset token.
	ResetPassword(ctx context.Context, resetToken, password string) error

	// SendPasswordReset sends reset password link to email.
	SendPasswordReset(ctx context.Context, redirectPath, email, token string) error

	// EnableUser logically enables the user identified with the provided ID
	EnableUser(ctx context.Context, token, id string) error

	// DisableUser logically disables the user identified with the provided ID
	DisableUser(ctx context.Context, token, id string) error

	// Backup returns admin, all users, and all OAuth identities. Only accessible by admin.
	Backup(ctx context.Context, token string) (User, []User, []Identity, error)

	// Restore restores users and OAuth identities from backup. Only accessible by admin.
	Restore(ctx context.Context, token string, admin User, users []User, identities []Identity) error

	PlatformInvites
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Email    string
	Status   string
	Metadata Metadata
	Order    string
	Dir      string
}

// UserPage contains a page of users.
type UserPage struct {
	Total uint64
	Users []User
}

type ConfigURLs struct {
	GoogleUserInfoURL   string
	GitHubUserInfoURL   string
	GitHubUserEmailsURL string
	RedirectLoginURL    string
}

var _ Service = (*usersService)(nil)

type usersService struct {
	users               UserRepository
	emailVerifications  EmailVerificationRepository
	invites             PlatformInvitesRepository
	identity            IdentityRepository
	inviteDuration      time.Duration
	emailVerifyEnabled  bool
	selfRegisterEnabled bool
	hasher              Hasher
	email               Emailer
	auth                protomfx.AuthServiceClient
	idProvider          uuid.IDProvider
	googleOAuth         oauth2.Config
	githubOAuth         oauth2.Config
	urls                ConfigURLs
}

// New instantiates the users service implementation
func New(users UserRepository, verifications EmailVerificationRepository, invites PlatformInvitesRepository, identity IdentityRepository, inviteDuration time.Duration, emailVerifyEnabled bool, selfRegisterEnabled bool, hasher Hasher, auth protomfx.AuthServiceClient, e Emailer, idp uuid.IDProvider, googleOAuth, githubOAuth oauth2.Config, urls ConfigURLs) Service {
	return &usersService{
		users:               users,
		emailVerifications:  verifications,
		invites:             invites,
		identity:            identity,
		inviteDuration:      inviteDuration,
		emailVerifyEnabled:  emailVerifyEnabled,
		selfRegisterEnabled: selfRegisterEnabled,
		hasher:              hasher,
		auth:                auth,
		email:               e,
		idProvider:          idp,
		googleOAuth:         googleOAuth,
		githubOAuth:         githubOAuth,
		urls:                urls,
	}
}

func (svc usersService) SelfRegister(ctx context.Context, user User, redirectPath string) (string, error) {
	if !svc.selfRegisterEnabled {
		return "", ErrSelfRegisterDisabled
	}

	_, err := svc.users.RetrieveByEmail(ctx, user.Email)
	if err != nil && !errors.Contains(err, dbutil.ErrNotFound) {
		return "", err
	}

	if err == nil {
		return "", dbutil.ErrConflict
	}

	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return "", errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	user.Password = hash

	if !svc.emailVerifyEnabled {
		userID, err := svc.idProvider.ID()
		if err != nil {
			return "", err
		}

		user.ID = userID
		user.Status = EnabledStatusKey

		if _, err := svc.users.Save(ctx, user); err != nil {
			return "", err
		}

		return user.ID, nil
	}

	token, err := svc.idProvider.ID()
	if err != nil {
		return "", err
	}

	verification := EmailVerification{
		User:      user,
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if _, err := svc.emailVerifications.Save(ctx, verification); err != nil {
		return "", err
	}

	go func() {
		// If an error occurs while attempting to send the e-mail including confirmation token to the user,
		// abort the process i.e. remove the pending Verification from the database.
		if err := svc.email.SendEmailVerification([]string{user.Email}, redirectPath, token); err != nil {
			svc.emailVerifications.Remove(ctx, token)
		}
	}()

	return token, nil
}

func (svc usersService) RegisterByInvite(ctx context.Context, user User, inviteID, orgInviteRedirectPath string) (string, error) {
	// Make sure user with same e-mail isn't registered already
	_, err := svc.users.RetrieveByEmail(ctx, user.Email)
	if err != nil && !errors.Contains(err, dbutil.ErrNotFound) {
		return "", err
	}

	if err == nil {
		return "", dbutil.ErrConflict
	}

	err = svc.ValidatePlatformInvite(ctx, inviteID, user.Email)
	if err != nil {
		return "", err
	}

	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return "", errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	user.Password = hash

	userID, err := svc.idProvider.ID()
	if err != nil {
		return "", err
	}

	user.ID = userID
	user.Status = EnabledStatusKey

	if _, err := svc.users.Save(ctx, user); err != nil {
		return "", err
	}

	// gRPC call to activate dormant Org Invites associated with this particular Platform Invite
	dormantOrgInvitesReq := &protomfx.ActivateOrgInviteReq{
		PlatformInviteID: inviteID,
		UserID:           userID,
		RedirectPath:     orgInviteRedirectPath,
	}

	if _, err := svc.auth.ActivateOrgInvite(ctx, dormantOrgInvitesReq); err != nil {
		return "", err
	}

	return user.ID, nil
}

func (svc usersService) VerifyEmail(ctx context.Context, confirmationToken string) (string, error) {
	verification, err := svc.emailVerifications.RetrieveByToken(ctx, confirmationToken)
	if err != nil {
		if errors.Contains(err, dbutil.ErrNotFound) {
			return "", errors.Wrap(errors.ErrAuthentication, err)
		}

		return "", err
	}

	if time.Now().After(verification.ExpiresAt) {
		if err := svc.emailVerifications.Remove(ctx, confirmationToken); err != nil {
			return "", err
		}

		return "", ErrEmailVerificationExpired
	}

	userID, err := svc.idProvider.ID()
	if err != nil {
		return "", err
	}

	verification.User.ID = userID
	verification.User.Status = EnabledStatusKey

	if _, err := svc.users.Save(ctx, verification.User); err != nil {
		return "", err
	}

	if err := svc.emailVerifications.Remove(ctx, confirmationToken); err != nil {
		return "", err
	}

	return userID, nil
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

	uid, err := svc.idProvider.ID()
	if err != nil {
		return err
	}
	user.ID = uid

	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return errors.Wrap(dbutil.ErrMalformedEntity, err)
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

	uid, err := svc.idProvider.ID()
	if err != nil {
		return "", err
	}

	user.ID = uid

	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return "", errors.Wrap(dbutil.ErrMalformedEntity, err)
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
	if dbUser.Password == "" {
		return "", errors.ErrAuthentication
	}
	if err := svc.hasher.Compare(user.Password, dbUser.Password); err != nil {
		return "", errors.Wrap(errors.ErrAuthentication, err)
	}
	return svc.issue(ctx, dbUser.ID, dbUser.Email, auth.LoginKey)
}

func (svc usersService) OAuthLogin(provider string) (state, verifier, redirectURL string, err error) {
	var oauthCfg oauth2.Config
	switch provider {
	case GoogleProvider:
		oauthCfg = svc.googleOAuth
	case GitHubProvider:
		oauthCfg = svc.githubOAuth
	default:
		return "", "", "", errors.ErrAuthorization
	}

	verifier = oauth2.GenerateVerifier()
	state, err = generateRandomState()
	if err != nil {
		return "", "", "", err
	}
	redirectURL = oauthCfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	return state, verifier, redirectURL, nil
}

func (svc usersService) OAuthCallback(ctx context.Context, provider, code, verifier string) (string, error) {
	var email, providerUserID string
	var err error

	switch provider {
	case GoogleProvider:
		email, providerUserID, err = svc.fetchGoogleUser(ctx, code, verifier)
	case GitHubProvider:
		email, providerUserID, err = svc.fetchGitHubUser(ctx, code, verifier)
	default:
		return "", errors.ErrAuthorization
	}

	if err != nil {
		return "", err
	}

	user, err := svc.handleIdentity(ctx, provider, email, providerUserID)
	if err != nil {
		return "", err
	}

	token, err := svc.issue(ctx, user.ID, user.Email, auth.LoginKey)
	if err != nil {
		return "", err
	}

	redirectURL := fmt.Sprintf("%s?token=%s", svc.urls.RedirectLoginURL, token)
	return redirectURL, nil
}

func (svc usersService) fetchGoogleUser(ctx context.Context, code, verifier string) (string, string, error) {
	oauthToken, err := svc.googleOAuth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return "", "", err
	}
	client := svc.googleOAuth.Client(ctx, oauthToken)
	resp, err := client.Get(svc.urls.GoogleUserInfoURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", errors.ErrAuthentication
	}

	var gUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return "", "", err
	}
	if gUser.Email == "" || gUser.ID == "" {
		return "", "", errors.ErrAuthentication
	}

	return gUser.Email, gUser.ID, nil
}

func (svc usersService) fetchGitHubUser(ctx context.Context, code, verifier string) (string, string, error) {
	oauthToken, err := svc.githubOAuth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return "", "", err
	}
	client := svc.githubOAuth.Client(ctx, oauthToken)

	var gUser struct {
		ID int64 `json:"id"`
	}
	resp, err := client.Get(svc.urls.GitHubUserInfoURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", errors.ErrAuthentication
	}

	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return "", "", err
	}
	providerUserID := strconv.FormatInt(gUser.ID, 10)

	resp2, err := client.Get(svc.urls.GitHubUserEmailsURL)
	if err != nil {
		return "", "", err
	}
	defer resp2.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&emails); err != nil {
		return "", "", err
	}

	email := ""
	for _, e := range emails {
		if e.Primary && e.Verified {
			email = e.Email
			break
		}
	}
	if email == "" {
		return "", "", errors.ErrAuthentication
	}

	return email, providerUserID, nil
}

func (svc usersService) handleIdentity(ctx context.Context, provider, email, providerUserID string) (User, error) {
	identity, err := svc.identity.Retrieve(ctx, provider, providerUserID)
	if err != nil && !errors.Contains(err, dbutil.ErrNotFound) {
		return User{}, err
	}

	var user User

	if identity.UserID != "" {
		user, err = svc.users.RetrieveByID(ctx, identity.UserID)
		if err != nil {
			return User{}, err
		}

		if user.Email != email {
			user.Email = email
			if err := svc.users.Update(ctx, user); err != nil {
				return User{}, err
			}
		}
	} else {
		user, err = svc.users.RetrieveByEmail(ctx, email)
		if err != nil {
			if errors.Contains(err, dbutil.ErrNotFound) {
				uid, err := svc.idProvider.ID()
				if err != nil {
					return User{}, err
				}
				user = User{
					ID:     uid,
					Email:  email,
					Status: EnabledStatusKey,
				}
				if _, err := svc.users.Save(ctx, user); err != nil {
					return User{}, err
				}
			} else {
				return User{}, err
			}
		}

		newIdentity := Identity{
			UserID:         user.ID,
			Provider:       provider,
			ProviderUserID: providerUserID,
		}
		if err := svc.identity.Save(ctx, newIdentity); err != nil {
			return User{}, err
		}
	}

	return user, nil
}

func (svc usersService) ViewUser(ctx context.Context, token, id string) (User, error) {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return User{}, err
	}

	if err := svc.isAdmin(ctx, token); err != nil {
		if user.id != id {
			return User{}, errors.ErrAuthorization
		}
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return User{}, errors.Wrap(dbutil.ErrNotFound, err)
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

func (svc usersService) ListUsersByIDs(ctx context.Context, ids []string, pm PageMetadata) (UserPage, error) {
	pm.Status = EnabledStatusKey
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

func (svc usersService) Backup(ctx context.Context, token string) (User, []User, []Identity, error) {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return User{}, []User{}, []Identity{}, err
	}

	if err := svc.isAdmin(ctx, token); err != nil {
		return User{}, []User{}, []Identity{}, err
	}

	users, err := svc.users.BackupAll(ctx)
	if err != nil {
		return User{}, []User{}, []Identity{}, err
	}

	var admin User
	for i, u := range users {
		if u.Email == user.email {
			admin = u
			users = append(users[:i], users[i+1:]...)
			break
		}
	}

	identities, err := svc.identity.BackupAll(ctx)
	if err != nil {
		return User{}, []User{}, []Identity{}, err
	}

	return admin, users, identities, nil
}

func (svc usersService) Restore(ctx context.Context, token string, admin User, users []User, identities []Identity) error {
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

	for _, identity := range identities {
		if err := svc.identity.Save(ctx, identity); err != nil {
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

func (svc usersService) GenerateResetToken(ctx context.Context, email, redirectPath string) error {
	user, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil || user.Email == "" {
		return dbutil.ErrNotFound
	}
	t, err := svc.issue(ctx, user.ID, user.Email, auth.RecoveryKey)
	if err != nil {
		return errors.Wrap(ErrRecoveryToken, err)
	}

	go func() {
		svc.SendPasswordReset(ctx, redirectPath, email, t)
	}()

	return nil
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
		return dbutil.ErrNotFound
	}

	password, err = svc.hasher.Hash(password)
	if err != nil {
		return err
	}
	return svc.users.UpdatePassword(ctx, ir.email, password)
}

func (svc usersService) ChangePassword(ctx context.Context, token, email, password, oldPassword string) error {
	ir, err := svc.identify(ctx, token)
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	var userEmail string

	switch {
	// Admin changes password for another user
	case oldPassword == "" && email != "":
		if err := svc.isAdmin(ctx, token); err != nil {
			return err
		}
		userEmail = email

	// User changes their own password
	case oldPassword != "" && email == "":
		u := User{
			Email:    ir.email,
			Password: oldPassword,
		}
		if _, err := svc.Login(ctx, u); err != nil {
			return errors.ErrInvalidPassword
		}
		userEmail = ir.email

	default:
		return errors.ErrAuthentication
	}

	hashedPassword, err := svc.hasher.Hash(password)
	if err != nil {
		return err
	}

	return svc.users.UpdatePassword(ctx, userEmail, hashedPassword)
}

func (svc usersService) SendPasswordReset(_ context.Context, redirectPath, email, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, redirectPath, token)
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
		return errors.Wrap(dbutil.ErrNotFound, err)
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
		return "", errors.Wrap(dbutil.ErrNotFound, err)
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
		Subject: auth.RootSub,
	}

	if _, err := svc.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
