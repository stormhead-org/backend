package grpcauthorization

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	eventpkg "github.com/stormhead-org/backend/internal/event"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
	protopkg "github.com/stormhead-org/backend/internal/proto"
	securitypkg "github.com/stormhead-org/backend/internal/security"
)

func (s *AuthorizationServer) Register(ctx context.Context, req *protopkg.RegisterRequest) (*protopkg.RegisterResponse, error) {
	if req.Email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	// Validate password complexity
	if len(req.Password) < 12 {
		return nil, status.Errorf(codes.InvalidArgument, "password must be at least 12 characters long")
	}

	// Check if password has been pwned
	isPwned, err := s.hibp.IsPasswordPwned(req.Password)
	if err != nil {
		s.log.Error("failed to check password against HIBP", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to validate password")
	}
	if isPwned {
		return nil, status.Errorf(codes.InvalidArgument, "password has been pwned, please choose a different one")
	}

	// Validate slug (assuming email is used as slug for now)
	_, err = s.database.SelectUserBySlug(
		req.Email,
	)
	if err != gorm.ErrRecordNotFound {
		return nil, status.Errorf(codes.InvalidArgument, "slug already exist")
	}

	// Validate name (assuming email is used as name for now)
	_, err = s.database.SelectUserByName(
		req.Email,
	)
	if err != gorm.ErrRecordNotFound {
		return nil, status.Errorf(codes.InvalidArgument, "name already exist")
	}

	// Validate email
	_, err = s.database.SelectUserByEmail(
		req.Email,
	)
	if err != gorm.ErrRecordNotFound {
		return nil, status.Errorf(codes.InvalidArgument, "email already exist")
	}

	// Salt password
	salt := securitypkg.GenerateSalt()

	hash, err := securitypkg.HashPassword(
		req.Password,
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
		Slug:              req.Email, // Assuming email is used as slug for now, will be updated later
		Name:              req.Email, // Assuming email is used as name for now, will be updated later
		Email:             req.Email,
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

	return &protopkg.RegisterResponse{
			UserId: user.ID.String(),
		},
		nil
}
