package service

import (
	"auth_service/internal/core/domain"
	repo "auth_service/internal/core/port/repo"
	service "auth_service/internal/core/port/service"
	dto "auth_service/internal/core/port/service/dto"
	"context"
	"errors"
	"errs"
	"jwtutils"
	"logs"
	"time"
)

type authService struct {
	userRepo    repo.UserRepository
	adminRepo   repo.AdminRepository
	sessionRepo repo.SessionRepository
	jwtService  *jwtutils.JWTService
}

func NewAuthService(userRepo repo.UserRepository, adminRepo repo.AdminRepository, sessionRepo repo.SessionRepository, jwtService *jwtutils.JWTService) service.AuthService {
	return &authService{userRepo: userRepo,
		adminRepo:   adminRepo,
		sessionRepo: sessionRepo,
		jwtService:  jwtService,
	}
}

func (s *authService) LoginUser(ctx context.Context, email, password, ipAddress, deviceInfo string) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			logs.Error(err)
			return nil, errs.NewUnauthorizedError("invalid credentials")
		}
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	err = user.CheckPassword(password)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnauthorizedError("incorrect password")
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
		UserID:       &user.ID,
		AdminID:      nil,
		RefreshToken: refreshToken,
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		ExpiredAt:    time.Now().Add(7 * 24 * time.Hour), // ต้องตรงกับ JWT Config
		IsRevoked:    false,
	}

	if err = s.sessionRepo.CreateSession(ctx, session); err != nil {
		logs.Error(err)
		return nil, err
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil

}

func (s *authService) LoginAdmin(ctx context.Context, username, password, ipAddress, deviceInfo string) (*dto.LoginResponse, error) {
	admin, err := s.adminRepo.FindByUserName(ctx, username)

	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	if admin == nil {
		return nil, errs.NewUnauthorizedError("invalid credentials")
	}

	err = admin.CheckPassword(password)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnauthorizedError("incorrect password")
	}

	accessToken, err := s.jwtService.GenerateToken(admin.ID, "admin", jwtutils.AccessToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("access generate token error")
	}

	refreshToken, err := s.jwtService.GenerateToken(admin.ID, "admin", jwtutils.RefreshToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("refresh generate token error")
	}

	session := &domain.Session{
		AdminID:      &admin.ID,
		UserID:       nil,
		RefreshToken: refreshToken,
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		ExpiredAt:    time.Now().Add(7 * 24 * time.Hour),
		IsRevoked:    false,
	}

	if err = s.sessionRepo.CreateSession(ctx, session); err != nil {
		logs.Error(err)
		return nil, err
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		logs.Error(err)
		return errs.NewNotFoundError("session not found")
	}

	if session.IsRevoked {
		return errs.NewValidationError("token is already revoked")
	}

	if err := s.sessionRepo.RevokeSession(ctx, refreshToken); err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	return nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dto.LoginResponse, error) {
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

	// ดึงค่า ID ออกมาจาก Session ให้ถูกต้องก่อน
	var subjectID uint

	if session.UserID != nil {
		// ถ้าเป็นของ User ให้ดึงค่าจาก UserID
		subjectID = *session.UserID
	} else if session.AdminID != nil {
		// ถ้าเป็นของ Admin ให้ดึงค่าจาก AdminID
		subjectID = *session.AdminID
	} else {
		// กรณีข้อมูลในฐานข้อมูลพัง (ไม่มีทั้งคู่)
		return nil, errs.NewUnexpectedError()
	}

	// ส่ง subjectID ที่ได้เข้าไปแทน
	newAccessToken, err := s.jwtService.GenerateToken(subjectID, claims.Role, jwtutils.AccessToken)

	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("access generate token error")
	}

	return &dto.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken,
	}, nil
}
