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

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"golang.org/x/oauth2"
)

const (
	EnabledStatusKey  = domain.EnabledStatusKey
	DisabledStatusKey = domain.DisabledStatusKey
	AllStatusKey      = domain.AllStatusKey
	GoogleProvider    = "google"
	GitHubProvider    = "github"
)

var (
	// ErrRecoveryToken indicates error in generating password recovery token.
	ErrRecoveryToken = errors.New("failed to generate password recovery token")

	// ErrAlreadyEnabledUser indicates the user is already enabled.
	ErrAlreadyEnabledUser = errors.New("the user is already enabled")

	// ErrAlreadyDisabledUser indicates the user is already disabled.
	ErrAlreadyDisabledUser = errors.New("the user is already disabled")

	// ErrEmailVerificationExpired indicates that the e-mail verification token has expired.
	ErrEmailVerificationExpired = errors.New("e-mail verification token expired")

	// ErrSelfRegisterDisabled indicates that self-registration is disabled in the service config.
	ErrSelfRegisterDisabled = errors.New("self register disabled")

	// ErrInvalidAdminCredentials indicates that the configured root admin e-mail or password is invalid.
	ErrInvalidAdminCredentials = errors.New("invalid root admin e-mail or password")

	// ErrPlatformRoleAssigned indicates that the user already has a platform role.
	ErrPlatformRoleAssigned = errors.New("the user already has a platform role")

	// ErrNotPlatformAdmin indicates that the user does not have the platform admin role.
	ErrNotPlatformAdmin = errors.New("the user does not have the platform admin role")

	// ErrDisabledUser indicates that the operation is not allowed for a disabled user.
	ErrDisabledUser = errors.New("the user is disabled")
)

type (
	PageMetadata = domain.UsersPageMetadata
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

	// RegisterAdmin creates root admin account, validating e-mail and password on creation.
	// Existing user with given e-mail is assigned root admin role and password is ignored.
	RegisterAdmin(ctx context.Context, user User) error

	// Login authenticates the user given its credentials. Successful
	// authentication generates new access token. Failed invocations are
	// identified by the non-nil error values in the response.
	Login(ctx context.Context, user User) (string, error)

	// RefreshToken issues a new access token to the bearer of a still-valid one,
	// extending the session without re-sending credentials.
	RefreshToken(ctx context.Context, token string) (string, error)

	// OAuthLogin returns the URL to initiate OAuth login.
	OAuthLogin(provider string) (data OAuthLoginData, err error)

	// OAuthCallback exchanges the OAuth code for user info and logs in/creates the user.
	OAuthCallback(ctx context.Context, data OAuthCallbackData) (string, error)

	// ViewUser retrieves user info for a given user ID and an authorized token.
	ViewUser(ctx context.Context, token, id string) (User, error)

	// ViewProfile retrieves user info for a given token.
	ViewProfile(ctx context.Context, token string) (User, error)

	// ListUsers retrieves users list for a valid admin token.
	ListUsers(ctx context.Context, token string, pm PageMetadata) (UsersPage, error)

	// ListUsersByIDs retrieves users list for the given IDs.
	ListUsersByIDs(ctx context.Context, ids []string, pm PageMetadata) (UsersPage, error)

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

	// Impersonate issues a login token for the user identified with the provided ID,
	// on behalf of the platform admin identified by the provided token.
	// Only the root admin is allowed to impersonate a platform admin.
	Impersonate(ctx context.Context, token, id string) (string, error)

	// AssignAdminRole assigns the platform admin role to the user with the provided ID.
	// Only the root admin is allowed to assign it.
	AssignAdminRole(ctx context.Context, token, id string) error

	// UnassignAdminRole unassigns the platform admin role from the user with the provided ID.
	// Only the root admin is allowed to unassign it.
	UnassignAdminRole(ctx context.Context, token, id string) error

	// Backup returns admin, all users, and all OAuth identities. Only accessible by root admin.
	Backup(ctx context.Context, token string) (User, []User, []Identity, error)

	// Restore restores users and OAuth identities from backup. Only accessible by root admin.
	Restore(ctx context.Context, token string, admin User, users []User, identities []Identity) error

	PlatformInvites
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
	legacyHasher        Hasher
	email               Emailer
	auth                domain.AuthClient
	idProvider          uuid.IDProvider
	googleOAuth         oauth2.Config
	githubOAuth         oauth2.Config
	urls                ConfigURLs
	passMinLength       int
	logger              logger.Logger
}

// Config holds configuration values for the users service.
type Config struct {
	InviteDuration      time.Duration
	EmailVerifyEnabled  bool
	SelfRegisterEnabled bool
	GoogleOAuth         oauth2.Config
	GitHubOAuth         oauth2.Config
	URLs                ConfigURLs
	PassMinLength       int
}

// New instantiates the users service implementation
func New(users UserRepository, verifications EmailVerificationRepository, invites PlatformInvitesRepository, identity IdentityRepository, hasher, legacyHasher Hasher, auth domain.AuthClient, e Emailer, idp uuid.IDProvider, logger logger.Logger, c Config) Service {
	return &usersService{
		users:               users,
		emailVerifications:  verifications,
		invites:             invites,
		identity:            identity,
		inviteDuration:      c.InviteDuration,
		emailVerifyEnabled:  c.EmailVerifyEnabled,
		selfRegisterEnabled: c.SelfRegisterEnabled,
		hasher:              hasher,
		legacyHasher:        legacyHasher,
		auth:                auth,
		email:               e,
		idProvider:          idp,
		googleOAuth:         c.GoogleOAuth,
		githubOAuth:         c.GitHubOAuth,
		urls:                c.URLs,
		passMinLength:       c.PassMinLength,
		logger:              logger,
	}
}

func (svc usersService) SelfRegister(ctx context.Context, user User, redirectPath string) (string, error) {
	if !svc.selfRegisterEnabled {
		return "", ErrSelfRegisterDisabled
	}

	if err := ValidatePassword(user.Password, user.Email, svc.passMinLength); err != nil {
		return "", err
	}

	_, err := svc.users.RetrieveByEmail(ctx, user.Email)
	if err != nil && !errors.Contains(err, dbutil.ErrNotFound) {
		return "", err
	}

	if err == nil {
		return "", dbutil.ErrConflict
	}

	hash, err := svc.hashPassword(user.Password)
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

	if err := ValidatePassword(user.Password, user.Email, svc.passMinLength); err != nil {
		return "", err
	}

	hash, err := svc.hashPassword(user.Password)
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
	if err := svc.auth.ActivateOrgInvite(ctx, inviteID, userID, orgInviteRedirectPath); err != nil {
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
		role, err := svc.auth.RetrieveRole(ctx, u.ID)
		if err != nil {
			return err
		}

		switch role {
		case domain.RoleRootAdmin:
			return nil
		case domain.RoleAdmin:
			if err := svc.auth.UnassignRole(ctx, u.ID); err != nil {
				return err
			}
		}

		if err := svc.auth.AssignRole(ctx, u.ID, domain.RoleRootAdmin); err != nil {
			return err
		}

		return nil
	}

	if err := user.Validate(); err != nil {
		return errors.Wrap(ErrInvalidAdminCredentials, err)
	}

	if err := ValidatePassword(user.Password, user.Email, svc.passMinLength); err != nil {
		return errors.Wrap(ErrInvalidAdminCredentials, err)
	}

	uid, err := svc.idProvider.ID()
	if err != nil {
		return err
	}
	user.ID = uid

	hash, err := svc.hashPassword(user.Password)
	if err != nil {
		return errors.Wrap(dbutil.ErrMalformedEntity, err)
	}
	user.Password = hash

	user.Status = EnabledStatusKey

	if _, err := svc.users.Save(ctx, user); err != nil {
		return err
	}

	if err := svc.auth.AssignRole(ctx, user.ID, domain.RoleRootAdmin); err != nil {
		return err
	}

	return nil
}

func (svc usersService) Register(ctx context.Context, token string, user User) (string, error) {
	if err := svc.canAccessPlatform(ctx, token); err != nil {
		return "", err
	}

	if err := ValidatePassword(user.Password, user.Email, svc.passMinLength); err != nil {
		return "", err
	}

	uid, err := svc.idProvider.ID()
	if err != nil {
		return "", err
	}

	user.ID = uid

	hash, err := svc.hashPassword(user.Password)
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
	if err := svc.comparePassword(ctx, dbUser, user.Password); err != nil {
		return "", errors.Wrap(errors.ErrAuthentication, err)
	}
	return svc.issue(ctx, dbUser.ID, dbUser.Email, domain.LoginKey)
}

func (svc usersService) RefreshToken(ctx context.Context, token string) (string, error) {
	t, err := svc.auth.RefreshToken(ctx, token)
	if err != nil {
		return "", err
	}

	return t, nil
}

func (svc usersService) OAuthLogin(provider string) (data OAuthLoginData, err error) {
	var oauthCfg oauth2.Config
	switch provider {
	case GoogleProvider:
		oauthCfg = svc.googleOAuth
	case GitHubProvider:
		oauthCfg = svc.githubOAuth
	default:
		return OAuthLoginData{}, apiutil.ErrInvalidProvider
	}

	data.Verifier = oauth2.GenerateVerifier()
	data.State, err = generateRandomState()
	if err != nil {
		return OAuthLoginData{}, err
	}
	data.RedirectURL = oauthCfg.AuthCodeURL(data.State, oauth2.S256ChallengeOption(data.Verifier))
	return data, nil
}

func (svc usersService) OAuthCallback(ctx context.Context, data OAuthCallbackData) (string, error) {
	var email, providerUserID string
	var err error

	switch data.Provider {
	case GoogleProvider:
		email, providerUserID, err = svc.fetchGoogleUser(ctx, data.Code, data.Verifier)
	case GitHubProvider:
		email, providerUserID, err = svc.fetchGitHubUser(ctx, data.Code, data.Verifier)
	default:
		return "", apiutil.ErrInvalidProvider
	}

	if err != nil {
		return "", err
	}

	user, err := svc.handleIdentity(ctx, data.Provider, email, providerUserID, data.InviteID, data.RedirectPath)
	if err != nil {
		return "", err
	}

	token, err := svc.issue(ctx, user.ID, user.Email, domain.LoginKey)
	if err != nil {
		return "", err
	}

	redirectURL := fmt.Sprintf("%s#token=%s", svc.urls.RedirectLoginURL, token)
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
	if gUser.ID == 0 {
		return "", "", errors.ErrAuthentication
	}
	providerUserID := strconv.FormatInt(gUser.ID, 10)

	resp2, err := client.Get(svc.urls.GitHubUserEmailsURL)
	if err != nil {
		return "", "", err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		return "", "", errors.ErrAuthentication
	}

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

// handleIdentity resolves or creates a User for an OAuth login flow.
// It first checks for an existing provider identity. If found, it returns the
// linked user (syncing their e-mail if it changed). Otherwise, it looks up or
// creates a user by e-mail, handling invite validation if needed, and saves a
// new identity record linking the provider to that user.
func (svc usersService) handleIdentity(ctx context.Context, provider, email, providerUserID, inviteID, redirectPath string) (User, error) {
	identity, err := svc.identity.Retrieve(ctx, provider, providerUserID)
	if err != nil && !errors.Contains(err, dbutil.ErrNotFound) {
		return User{}, err
	}

	if identity.UserID != "" {
		user, err := svc.users.RetrieveByID(ctx, identity.UserID)
		if err != nil {
			return User{}, err
		}
		if user.Status != EnabledStatusKey {
			return User{}, errors.ErrAuthentication
		}
		if user.Email != email {
			user.Email = email
			if err := svc.users.Update(ctx, user); err != nil {
				return User{}, err
			}
		}
		return user, nil
	}

	user, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil {
		if !errors.Contains(err, dbutil.ErrNotFound) {
			return User{}, err
		}
		if !svc.selfRegisterEnabled && inviteID == "" {
			return User{}, ErrSelfRegisterDisabled
		}
		if inviteID != "" {
			if err := svc.ValidatePlatformInvite(ctx, inviteID, email); err != nil {
				return User{}, err
			}
		}
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
		if inviteID != "" {
			if err := svc.auth.ActivateOrgInvite(ctx, inviteID, user.ID, redirectPath); err != nil {
				return User{}, err
			}
		}
	}

	newIdentity := Identity{
		UserID:         user.ID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}
	if err := svc.identity.Save(ctx, newIdentity); err != nil {
		if !errors.Contains(err, dbutil.ErrConflict) {
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

	if err := svc.canAccessPlatform(ctx, token); err != nil {
		if user.id != id {
			return User{}, errors.ErrAuthorization
		}
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return User{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	role, err := svc.auth.RetrieveRole(ctx, id)
	if err != nil {
		return User{}, err
	}

	return User{
		ID:       id,
		Email:    dbUser.Email,
		Password: "",
		Metadata: dbUser.Metadata,
		Status:   dbUser.Status,
		Role:     role,
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

	role, err := svc.auth.RetrieveRole(ctx, ir.id)
	if err != nil {
		return User{}, err
	}

	return User{
		ID:       dbUser.ID,
		Email:    ir.email,
		Metadata: dbUser.Metadata,
		Role:     role,
	}, nil
}

func (svc usersService) ListUsers(ctx context.Context, token string, pm PageMetadata) (UsersPage, error) {
	if err := svc.canAccessPlatform(ctx, token); err != nil {
		return UsersPage{}, err
	}

	up, err := svc.users.RetrieveByIDs(ctx, nil, pm)
	if err != nil {
		return UsersPage{}, err
	}

	if len(up.Users) == 0 {
		return up, nil
	}

	ids := make([]string, 0, len(up.Users))
	for _, u := range up.Users {
		ids = append(ids, u.ID)
	}

	roles, err := svc.auth.RetrieveRoles(ctx, ids)
	if err != nil {
		return UsersPage{}, err
	}

	for i := range up.Users {
		up.Users[i].Role = roles[up.Users[i].ID]
	}

	return up, nil
}

func (svc usersService) ListUsersByIDs(ctx context.Context, ids []string, pm PageMetadata) (UsersPage, error) {
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

	if err := svc.canAccessAsRoot(ctx, user.id); err != nil {
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
	ir, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canAccessAsRoot(ctx, ir.id); err != nil {
		return err
	}

	if err := svc.restoreAdmin(ctx, ir, admin); err != nil {
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
	return svc.users.UpdateUserMetadata(ctx, user)
}

func (svc usersService) GenerateResetToken(ctx context.Context, email, redirectPath string) error {
	user, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil || user.Email == "" {
		return dbutil.ErrNotFound
	}
	t, err := svc.issue(ctx, user.ID, user.Email, domain.RecoveryKey)
	if err != nil {
		return errors.Wrap(ErrRecoveryToken, err)
	}

	go func() {
		svc.SendPasswordReset(ctx, redirectPath, email, t)
	}()

	return nil
}

func (svc usersService) ResetPassword(ctx context.Context, resetToken, password string) error {
	identity, err := svc.auth.IdentifyRecovery(ctx, resetToken)
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}
	u, err := svc.users.RetrieveByID(ctx, identity.ID)
	if err != nil {
		return err
	}
	if u.Email == "" {
		return dbutil.ErrNotFound
	}

	if err := ValidatePassword(password, u.Email, svc.passMinLength); err != nil {
		return err
	}

	password, err = svc.hashPassword(password)
	if err != nil {
		return err
	}
	if err := svc.users.UpdatePassword(ctx, identity.Email, password); err != nil {
		return err
	}

	return svc.auth.RevokeSessions(ctx, identity.ID)
}

func (svc usersService) ChangePassword(ctx context.Context, token, email, password, oldPassword string) error {
	ir, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	var userEmail, userID string

	switch {
	// Admin changes password for another user
	case oldPassword == "" && email != "":
		if err := svc.canAccessPlatform(ctx, token); err != nil {
			return err
		}
		u, err := svc.users.RetrieveByEmail(ctx, email)
		if err != nil {
			return err
		}
		if err := svc.canModifyUser(ctx, ir.id, u.ID); err != nil {
			return err
		}
		userEmail = email
		userID = u.ID

	case oldPassword != "" && email == "":
		u := User{
			Email:    ir.email,
			Password: oldPassword,
		}
		if _, err := svc.Login(ctx, u); err != nil {
			return errors.ErrInvalidPassword
		}
		userEmail = ir.email
		userID = ir.id

	default:
		return errors.ErrAuthentication
	}

	if err := ValidatePassword(password, userEmail, svc.passMinLength); err != nil {
		return err
	}

	hashedPassword, err := svc.hashPassword(password)
	if err != nil {
		return err
	}

	if err := svc.users.UpdatePassword(ctx, userEmail, hashedPassword); err != nil {
		return err
	}

	return svc.auth.RevokeSessions(ctx, userID)
}

func (svc usersService) hashPassword(password string) (string, error) {
	return svc.hasher.Hash(NormalizePassword(password))
}

func (svc usersService) comparePassword(ctx context.Context, dbUser User, password string) error {
	err := svc.hasher.Compare(NormalizePassword(password), dbUser.Password)
	if !errors.Contains(err, ErrUnsupportedHash) {
		return err
	}

	// Legacy hashes were created from the password as received, before normalization.
	if err := svc.legacyHasher.Compare(password, dbUser.Password); err != nil {
		return err
	}

	hash, err := svc.hashPassword(password)
	if err == nil {
		err = svc.users.UpdatePassword(ctx, dbUser.Email, hash)
	}
	if err != nil {
		// Login still succeeds; the hash is replaced on a later login.
		svc.logger.Warn(fmt.Sprintf("failed to replace legacy password hash of user %s: %s", dbUser.ID, err))
	}

	return nil
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

	return svc.auth.RevokeSessions(ctx, id)
}

func (svc usersService) Impersonate(ctx context.Context, token, id string) (string, error) {
	admin, err := svc.identify(ctx, token)
	if err != nil {
		return "", err
	}

	if admin.impersonator != nil || admin.id == id {
		return "", errors.ErrAuthorization
	}

	if err := svc.canAccessPlatform(ctx, token); err != nil {
		return "", err
	}

	target, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return "", errors.Wrap(dbutil.ErrNotFound, err)
	}

	if target.Status == DisabledStatusKey {
		return "", ErrDisabledUser
	}

	targetRole, err := svc.auth.RetrieveRole(ctx, target.ID)
	if err != nil {
		return "", err
	}

	if targetRole == domain.RoleRootAdmin {
		return "", errors.ErrAuthorization
	}

	if targetRole == domain.RoleAdmin {
		if err := svc.canAccessAsRoot(ctx, admin.id); err != nil {
			return "", err
		}
	}

	impersonator := domain.Identity{
		ID:    admin.id,
		Email: admin.email,
	}

	return svc.auth.IssueImpersonated(ctx, target.ID, target.Email, impersonator)
}

func (svc usersService) changeStatus(ctx context.Context, token, id, status string) error {
	ir, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canAccessPlatform(ctx, token); err != nil {
		return err
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(dbutil.ErrNotFound, err)
	}

	if err := svc.canModifyUser(ctx, ir.id, id); err != nil {
		return err
	}

	// A platform role grants access through API keys, which disabling the user does not revoke.
	if status == DisabledStatusKey {
		role, err := svc.auth.RetrieveRole(ctx, id)
		if err != nil {
			return err
		}

		if role != "" {
			return ErrPlatformRoleAssigned
		}
	}

	if dbUser.Status == status {
		if status == DisabledStatusKey {
			return ErrAlreadyDisabledUser
		}
		return ErrAlreadyEnabledUser
	}

	return svc.users.ChangeStatus(ctx, id, status)
}

func (svc usersService) AssignAdminRole(ctx context.Context, token, id string) error {
	ir, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canAccessAsRoot(ctx, ir.id); err != nil {
		return err
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(dbutil.ErrNotFound, err)
	}

	if dbUser.Status == DisabledStatusKey {
		return ErrDisabledUser
	}

	role, err := svc.auth.RetrieveRole(ctx, id)
	if err != nil {
		return err
	}

	if role != "" {
		return ErrPlatformRoleAssigned
	}

	return svc.auth.AssignRole(ctx, id, domain.RoleAdmin)
}

func (svc usersService) UnassignAdminRole(ctx context.Context, token, id string) error {
	ir, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canAccessAsRoot(ctx, ir.id); err != nil {
		return err
	}

	role, err := svc.auth.RetrieveRole(ctx, id)
	if err != nil {
		return err
	}

	if role == domain.RoleRootAdmin {
		return errors.ErrAuthorization
	}

	if role != domain.RoleAdmin {
		return ErrNotPlatformAdmin
	}

	// Sessions are revoked first so that a retry after a failed revocation still finds the role.
	if err := svc.auth.RevokeSessions(ctx, id); err != nil {
		return err
	}

	return svc.auth.UnassignRole(ctx, id)
}

// restoreAdmin assigns the root role only to the configured root admin.
func (svc usersService) restoreAdmin(ctx context.Context, root userIdentity, admin User) error {
	if admin.Email != root.email {
		_, err := svc.users.Save(ctx, admin)
		return err
	}

	if err := svc.users.UpdateUserMetadata(ctx, admin); err != nil {
		return err
	}

	role, err := svc.auth.RetrieveRole(ctx, admin.ID)
	if err != nil {
		return err
	}

	if role != domain.RoleRootAdmin {
		if err := svc.auth.AssignRole(ctx, admin.ID, domain.RoleRootAdmin); err != nil {
			return err
		}
	}

	if admin.ID == root.id {
		return nil
	}

	// The root admin user now has the ID from the backup, so its previous ID no longer holds the role.
	prevRole, err := svc.auth.RetrieveRole(ctx, root.id)
	if err != nil {
		return err
	}

	if prevRole == "" {
		return nil
	}

	return svc.auth.UnassignRole(ctx, root.id)
}

// Auth helpers
func (svc usersService) issue(ctx context.Context, id, email string, keyType uint32) (string, error) {
	key, err := svc.auth.Issue(ctx, id, email, keyType)
	if err != nil {
		return "", errors.Wrap(dbutil.ErrNotFound, err)
	}
	return key, nil
}

type userIdentity struct {
	id           string
	email        string
	impersonator *domain.Identity
}

func (svc usersService) identify(ctx context.Context, token string) (userIdentity, error) {
	identity, err := svc.auth.Identify(ctx, token)
	if err != nil {
		return userIdentity{}, err
	}

	user := userIdentity{
		id:           identity.ID,
		email:        identity.Email,
		impersonator: identity.Impersonator,
	}

	return user, nil
}

func (svc usersService) canAccessPlatform(ctx context.Context, token string) error {
	if err := svc.auth.Authorize(ctx, domain.AuthzReq{Token: token, Subject: domain.RootSub}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}

func (svc usersService) canAccessAsRoot(ctx context.Context, userID string) error {
	role, err := svc.auth.RetrieveRole(ctx, userID)
	if err != nil {
		return err
	}

	if role != domain.RoleRootAdmin {
		return errors.ErrAuthorization
	}

	return nil
}

// canModifyUser restricts the modification of users with a platform role to the root admin.
func (svc usersService) canModifyUser(ctx context.Context, callerID, userID string) error {
	role, err := svc.auth.RetrieveRole(ctx, userID)
	if err != nil {
		return err
	}

	if role == "" {
		return nil
	}

	return svc.canAccessAsRoot(ctx, callerID)
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
