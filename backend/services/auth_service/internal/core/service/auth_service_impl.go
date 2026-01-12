package service

import (
	"auth_service/internal/core/domain"
	repo "auth_service/internal/core/port/repo"
	service "auth_service/internal/core/port/service"
	res "auth_service/internal/core/port/service/dto"
	"context"
	"errors"
	"errs"
	"jwtutils"
	"logs"
	"time"
)

type authService struct {
	userRepo repo.UserRepository
	// adminRepo   repo.AdminRepository
	sessionRepo repo.SessionRepository
	jwtService  *jwtutils.JWTService
}

// func NewAuthService(userRepo repo.UserRepository, adminRepo repo.AdminRepository, sessionRepo repo.SessionRepository, jwtService *jwtmiddleware.JWTService) service.AuthService {
// 	return &authService{userRepo: userRepo,
// 		adminRepo:   adminRepo,
// 		sessionRepo: sessionRepo,
// 		jwtService:  jwtService,
// 	}
// }

func NewAuthService(userRepo repo.UserRepository, sessionRepo repo.SessionRepository, jwtService *jwtutils.JWTService) service.AuthService {
	return &authService{userRepo: userRepo,
		sessionRepo: sessionRepo,
		jwtService:  jwtService,
	}
}

func (s *authService) LoginUser(ctx context.Context, email, password, ipAddress, deviceInfo string) (*res.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			logs.Error(err)
			return nil, errs.NewNotFoundError("user not found")
		}
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	err = user.CheckPassword(password)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnauthorizedError("incorrect old password")
	}

	accessToken, err := s.jwtService.GenerateToken(user.ID, "customer", jwtutils.AccessToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("access generate token error")
	}

	refreshToken, err := s.jwtService.GenerateToken(user.ID, "customer", jwtutils.RefreshToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("refresh generate token error")
	}

	// 4. Save Session
	session := &domain.Session{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		ExpiredAt:    time.Now().Add(7 * 24 * time.Hour), // ต้องตรงกับ JWT Config
		IsRevoked:    false,
	}

	if err := s.sessionRepo.CreateSession(ctx, session); err != nil {
		logs.Error(err)
		return nil, err
	}

	return &res.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil

}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*res.LoginResponse, error) {
	claims, err := s.jwtService.ValidateToken(refreshToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("invalid refresh token signature")
	}

	// Safety check: ป้องกัน panic ถ้า claims เป็น nil
	if claims == nil {
		return nil, errs.NewValidationError("invalid token claims")
	}

	if claims.Type != jwtutils.RefreshToken {
		return nil, errs.NewNotFoundError("invalid token type")
	}

	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	if session.IsRevoked {
		return nil, errs.NewNotFoundError("token has been revoked")
	}

	if time.Now().After(session.ExpiredAt) {
		return nil, errs.NewValidationError("refresh token expired")
	}

	newAccessToken, err := s.jwtService.GenerateToken(session.UserID, claims.Role, jwtutils.AccessToken)

	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("access generate token error")
	}

	return &res.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken,
	}, nil
}
