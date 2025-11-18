package user

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	clientpkg "github.com/stormhead-org/backend/internal/client"
	eventpkg "github.com/stormhead-org/backend/internal/event"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	securitypkg "github.com/stormhead-org/backend/internal/security"
	"github.com/stormhead-org/backend/internal/services"
)

type UserServiceImpl struct {
	log        *zap.Logger
	database   *ormpkg.PostgresClient
	broker     *eventpkg.KafkaClient
	hibpClient *clientpkg.HIBPClient
}

func NewUserService(log *zap.Logger, database *ormpkg.PostgresClient, broker *eventpkg.KafkaClient, hibpClient *clientpkg.HIBPClient) services.UserService {
	return &UserServiceImpl{
		log:        log,
		database:   database,
		broker:     broker,
		hibpClient: hibpClient,
	}
}

func (s *UserServiceImpl) Register(ctx context.Context, email, password string) (*ormpkg.User, error) {
	// Validate password complexity
	if len(password) < 12 {
		return nil, status.Errorf(codes.InvalidArgument, "password must be at least 12 characters long")
	}

	// Check if password has been pwned
	isPwned, err := s.hibpClient.IsPasswordPwned(password)
	if err != nil {
		s.log.Error("failed to check password against HIBP", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to validate password")
	}
	if isPwned {
		return nil, status.Errorf(codes.InvalidArgument, "password has been pwned, please choose a different one")
	}

	// Validate slug (assuming email is used as slug for now)
	_, err = s.database.SelectUserBySlug(
		email,
	)
	if err != gorm.ErrRecordNotFound {
		return nil, status.Errorf(codes.InvalidArgument, "slug already exist")
	}

	// Validate name (assuming email is used as name for now)
	_, err = s.database.SelectUserByName(
		email,
	)
	if err != gorm.ErrRecordNotFound {
		return nil, status.Errorf(codes.InvalidArgument, "name already exist")
	}

	// Validate email
	_, err = s.database.SelectUserByEmail(
		email,
	)
	if err != gorm.ErrRecordNotFound {
		return nil, status.Errorf(codes.InvalidArgument, "email already exist")
	}

	// Salt password
	salt := securitypkg.GenerateSalt()

	hash, err := securitypkg.HashPassword(
		password,
		salt,
	)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	// Check if this is the first user
	userCount, err := s.database.CountUsers()
	if err != nil {
		s.log.Error("failed to count users", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	isFirstUser := userCount == 0

	// Create user
	user := &ormpkg.User{
		Slug:              email, // Assuming email is used as slug for now, will be updated later
		Name:              email, // Assuming email is used as name for now, will be updated later
		Email:             email,
		Password:          hash,
		Salt:              salt,
		VerificationToken: securitypkg.GenerateToken(),
		IsVerified:        false,
		Reputation:        0,
		LastActivity:      time.Now(),
	}
	err = s.database.InsertUser(user)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	if isFirstUser {
		if err := s.database.UpdatePlatformOwner(user.ID); err != nil {
			s.log.Error("failed to set platform owner", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal error")
		}

		// Assign "platform owner" role to the first user
		ownerRole, err := s.database.SelectRoleByName("platform owner", nil) // nil for community_id for platform role
		if err == gorm.ErrRecordNotFound {
			// If "platform owner" role doesn't exist, create it (this should ideally be seeded)
			ownerRole = &ormpkg.Role{
				Name:        "platform owner",
				Color:       "#FFD700", // Gold color
				Type:        "platform",
				Permissions: []byte(`{"can_manage_platform": true}`), // Example permission
			}
			err = s.database.InsertRole(ownerRole)
			if err != nil {
				s.log.Error("failed to create platform owner role", zap.Error(err))
				return nil, status.Errorf(codes.Internal, "internal error")
			}
		} else if err != nil {
			s.log.Error("failed to select platform owner role", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal error")
		}

		userRole := &ormpkg.UserRole{
			UserID: user.ID,
			RoleID: ownerRole.ID,
		}
		err = s.database.InsertUserRole(userRole)
		if err != nil {
			s.log.Error("failed to assign platform owner role to first user", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal error")
		}
	}

	// Write message to broker
	err = s.broker.WriteMessage(
		ctx,
		eventpkg.AUTHORIZATION_REGISTER,
		eventpkg.AuthorizationRegisterMessage{
			ID: user.ID.String(),
		},
	)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return user, nil
}

func (s *UserServiceImpl) Login(ctx context.Context, email, password string) (*ormpkg.User, *ormpkg.Session, error) {
	// Get user from database
	user, err := s.database.SelectUserByEmail(
		email,
	)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "user not found")
	}
	if !user.IsVerified {
		return nil, nil, status.Errorf(codes.InvalidArgument, "user not verified")
	}

	err = securitypkg.ComparePasswords(
		user.Password,
		password,
		user.Salt,
	)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "password invalid")
	}

	// Obtain user agent and ip address
	userAgent := "unknown"
	m, ok := metadata.FromIncomingContext(ctx)
	if ok {
		userAgent = strings.Join(m["user-agent"], "")
	}

	ipAddress := "unknown"
	p, ok := peer.FromContext(ctx)
	if ok {
		parts := strings.Split(p.Addr.String(), ":")
		if len(parts) == 2 {
			ipAddress = parts[0]
		}
	}

	if userAgent == "unknown" || ipAddress == "unknown" {
		s.log.Error("internal error", zap.Error(err))
		return nil, nil, status.Errorf(codes.Internal, "internal error")
	}

	// Check existing sessions
	sessions, err := s.database.SelectSessionsByUserID(user.ID.String(), "", 0)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, nil, status.Errorf(codes.Internal, "internal error")
	}

	for _, session := range sessions {
		if session.IpAddress != ipAddress {
			continue
		}

		if session.UserAgent != userAgent {
			continue
		}

		s.log.Error("multiple login attempt from same client")
		// return nil, status.Errorf(codes.Internal, "multiple login attempt from same client")
	}

	// Create session
	session := ormpkg.Session{
		UserID:    user.ID,
		UserAgent: userAgent,
		IpAddress: ipAddress,
	}
	err = s.database.InsertSession(&session)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, nil, status.Errorf(codes.Internal, "internal error")
	}

	// Write message to broker
	err = s.broker.WriteMessage(
		ctx,
		eventpkg.AUTHORIZATION_LOGIN,
		eventpkg.AuthorizationLoginMessage{
			ID: user.ID.String(),
		},
	)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, nil, status.Errorf(codes.Internal, "internal error")
	}

	return user, &session, nil
}

func (s *UserServiceImpl) VerifyEmail(ctx context.Context, token string) error {
	user, err := s.database.SelectUserByVerificationToken(token)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "user not exist")
	}

	user.IsVerified = true
	if err := s.database.UpdateUser(user); err != nil {
		s.log.Error("internal error", zap.Error(err))
		return status.Errorf(codes.Internal, "internal error")
	}
	return nil
}

func (s *UserServiceImpl) RequestPasswordReset(ctx context.Context, email string) error {
	// Get user from database
	user, err := s.database.SelectUserByEmail(
		email,
	)
	if err != nil {
		// Always return success to prevent enumeration attacks, as specified in T043.
		// If the user is not found, we still return success but do not perform any reset actions.
		s.log.Warn("password reset requested for non-existent user", zap.String("email", email))
		return nil
	}
	if !user.IsVerified {
		// Always return success to prevent enumeration attacks.
		// If the user is not verified, we still return success but do not perform any reset actions.
		s.log.Warn("password reset requested for unverified user", zap.String("email", email), zap.String("userID", user.ID.String()))
		return nil
	}

	// Update user with reset token
	user.ResetToken = securitypkg.GenerateToken()
	expiresAt := time.Now().Add(time.Hour) // Token valid for 1 hour
	user.ResetTokenExpiresAt = &expiresAt

	err = s.database.UpdateUser(user)
	if err != nil {
		s.log.Error("failed to update user with reset token", zap.Error(err), zap.String("userID", user.ID.String()))
		return status.Errorf(codes.Internal, "internal error")
	}

	// Write message to broker for email sending
	err = s.broker.WriteMessage(
		ctx,
		eventpkg.AUTHORIZATION_REQUEST_PASSWORD_RESET,
		eventpkg.AuthorizationRequestPasswordReset{
			ID: user.ID.String(),
		},
	)
	if err != nil {
		s.log.Error("failed to write password reset event to broker", zap.Error(err), zap.String("userID", user.ID.String()))
		return status.Errorf(codes.Internal, "internal error")
	}

	return nil
}

func (s *UserServiceImpl) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	user, err := s.database.SelectUserByResetToken(token)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "user not exist or token is invalid")
	}

	if user.ResetTokenExpiresAt == nil || time.Now().After(*user.ResetTokenExpiresAt) {
		return status.Errorf(codes.InvalidArgument, "reset token expired or invalid")
	}

	if len(newPassword) < 12 {
		return status.Errorf(codes.InvalidArgument, "password must be at least 12 characters long")
	}

	isPwned, err := s.hibpClient.IsPasswordPwned(newPassword)
	if err != nil {
		s.log.Error("failed to check password against HIBP", zap.Error(err))
		return status.Errorf(codes.Internal, "failed to validate password")
	}
	if isPwned {
		return status.Errorf(codes.InvalidArgument, "password has been pwned, please choose a different one")
	}

	salt := securitypkg.GenerateSalt()
	hash, err := securitypkg.HashPassword(newPassword, salt)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return status.Errorf(codes.Internal, "internal error")
	}

	user.Password = hash
	user.Salt = salt
	user.ResetToken = ""
	user.ResetTokenExpiresAt = nil

	if err := s.database.UpdateUser(user); err != nil {
		s.log.Error("internal error", zap.Error(err))
		return status.Errorf(codes.Internal, "internal error")
	}

	if err := s.database.DeleteSessionsByUserID(user.ID.String()); err != nil {
		s.log.Error("failed to delete user sessions after password reset", zap.Error(err))
	}

	return nil
}

func (s *UserServiceImpl) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.database.SelectUserByID(userID.String())
	if err != nil {
		s.log.Error("failed to retrieve user for password change", zap.Error(err), zap.String("userID", userID.String()))
		return status.Errorf(codes.Internal, "internal error")
	}

	// Check old password
	err = securitypkg.ComparePasswords(
		user.Password,
		oldPassword,
		user.Salt,
	)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "old password invalid")
	}

	// Validate new password complexity
	if len(newPassword) < 12 {
		return status.Errorf(codes.InvalidArgument, "new password must be at least 12 characters long")
	}

	// Check if new password has been pwned
	isPwned, err := s.hibpClient.IsPasswordPwned(newPassword)
	if err != nil {
		s.log.Error("failed to check new password against HIBP", zap.Error(err))
		return status.Errorf(codes.Internal, "failed to validate new password")
	}
	if isPwned {
		return status.Errorf(codes.InvalidArgument, "new password has been pwned, please choose a different one")
	}

	// Salt new password
	salt := securitypkg.GenerateSalt()

	hash, err := securitypkg.HashPassword(
		newPassword,
		salt,
	)
	if err != nil {
		s.log.Error("failed to hash new password", zap.Error(err))
		return status.Errorf(codes.Internal, "internal error")
	}

	// Update user in database
	user.Password = hash
	user.Salt = salt

	err = s.database.UpdateUser(user)
	if err != nil {
		s.log.Error("failed to update user with new password", zap.Error(err), zap.String("userID", userID.String()))
		return status.Errorf(codes.Internal, "internal error")
	}

	// Revoke all active sessions for the user
	err = s.database.DeleteSessionsByUserID(userID.String())
	if err != nil {
		s.log.Error("failed to delete user sessions after password change", zap.Error(err), zap.String("userID", userID.String()))
		// Do not return an error here, as the password change was successful.
		// Logging the error is sufficient.
	}

	return nil
}

func (s *UserServiceImpl) GetSessionByID(ctx context.Context, sessionID string) (*ormpkg.Session, error) {
	session, err := s.database.SelectSessionByID(sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "database error")
	}
	return session, nil
}

func (s *UserServiceImpl) GetUserByID(ctx context.Context, userID string) (*ormpkg.User, error) {
	user, err := s.database.SelectUserByID(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "database error")
	}
	return user, nil
}

func (s *UserServiceImpl) GetCurrentSession(ctx context.Context, sessionID uuid.UUID) (*ormpkg.Session, error) {
	session, err := s.database.SelectSessionByID(sessionID.String())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "session not found")
		}
		s.log.Error("internal error", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error")
	}
	return session, nil
}

func (s *UserServiceImpl) ListActiveSessions(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*ormpkg.Session, string, error) {
	const SESSIONS_PER_PAGE = 10
	if limit <= 0 || limit > 50 {
		limit = SESSIONS_PER_PAGE
	}
	sessions, err := s.database.SelectSessionsByUserID(userID.String(), cursor, limit+1)
	if err != nil {
		s.log.Error("internal error", zap.Error(err))
		return nil, "", status.Errorf(codes.Internal, "internal error")
	}

	var nextCursor string
	if len(sessions) > limit {
		nextCursor = sessions[limit].ID.String()
		sessions = sessions[:limit]
	}

	return sessions, nextCursor, nil
}

func (s *UserServiceImpl) RevokeSession(ctx context.Context, currentSessionID uuid.UUID, sessionIDToRevoke uuid.UUID) error {
	// Get current user's session
	userSession, err := s.database.SelectSessionByID(currentSessionID.String())
	if err != nil {
		s.log.Error("failed to retrieve current user session", zap.Error(err), zap.String("sessionID", currentSessionID.String()))
		return status.Errorf(codes.Internal, "internal error")
	}

	// Get the session to be revoked
	requestedSession, err := s.database.SelectSessionByID(sessionIDToRevoke.String())
	if err != nil {
		s.log.Error("failed to retrieve requested session to revoke", zap.Error(err), zap.String("sessionID", sessionIDToRevoke.String()))
		return status.Errorf(codes.InvalidArgument, "session not found")
	}

	// Check if the session to be revoked belongs to the current user
	if userSession.UserID != requestedSession.UserID {
		s.log.Warn("attempt to revoke session belonging to another user",
			zap.String("currentUserID", userSession.UserID.String()),
			zap.String("requestedSessionUserID", requestedSession.UserID.String()),
			zap.String("sessionIDToRevoke", sessionIDToRevoke.String()))
		return status.Errorf(codes.PermissionDenied, "permission denied")
	}

	// Prevent revoking the current active session
	if currentSessionID == sessionIDToRevoke {
		s.log.Warn("attempt to revoke current active session", zap.String("sessionID", currentSessionID.String()))
		return status.Errorf(codes.InvalidArgument, "cannot revoke current active session")
	}

	// Delete session from database
	err = s.database.DeleteSession(requestedSession)
	if err != nil {
		s.log.Error("failed to delete session", zap.Error(err), zap.String("sessionID", sessionIDToRevoke.String()))
		return status.Errorf(codes.Internal, "internal error")
	}

	return nil
}
