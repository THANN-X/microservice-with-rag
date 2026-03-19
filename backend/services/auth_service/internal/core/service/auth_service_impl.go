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

// What: authService จัดการ login, logout และ token rotation
// Why:  แยก auth logic ออกจาก user/admin service เพราะมัน cross-cutting
//       (ทั้ง user และ admin login ผ่านที่เดียวกัน)
type authService struct {
	userRepo    repo.UserRepository
	adminRepo   repo.AdminRepository
	sessionRepo repo.SessionRepository
	jwtService  *jwtutils.JWTService
}

// What: constructor — inject dependencies ทั้งหมดที่ auth service ต้องการ
func NewAuthService(userRepo repo.UserRepository, adminRepo repo.AdminRepository, sessionRepo repo.SessionRepository, jwtService *jwtutils.JWTService) service.AuthService {
	return &authService{
		userRepo:    userRepo,
		adminRepo:   adminRepo,
		sessionRepo: sessionRepo,
		jwtService:  jwtService,
	}
}

// What: login flow สำหรับ user
//  1. หา user จาก email
//  2. ตรวจสอบ password
//  3. ออก access token + refresh token
//  4. บันทึก session ลง DB
//
// Why:  session ต้องบันทึกเพื่อให้ logout / revoke token ได้ทันที
func (s *authService) LoginUser(ctx context.Context, email, password, ipAddress, deviceInfo string) (*dto.LoginResponse, error) {
	// What: ค้นหา user ด้วย email — ถ้าไม่เจอให้ตอบ "invalid credentials" (ไม่บอกว่า email ไม่มี)
	// Why:  การบอกว่า email ไม่มีอยู่จะช่วย attacker enumerate accounts ได้
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			logs.Error(err)
			return nil, errs.NewUnauthorizedError("invalid credentials")
		}
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	// What: ตรวจสอบ password กับ bcrypt hash ที่เก็บไว้
	err = user.CheckPassword(password)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnauthorizedError("incorrect password")
	}

	// What: ออก short-lived access token (เช่น 15 นาที)
	accessToken, err := s.jwtService.GenerateToken(user.ID, "customer", jwtutils.AccessToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("access generate token error")
	}

	// What: ออก long-lived refresh token (7 วัน) ใช้ขอ access token ใหม่
	refreshToken, err := s.jwtService.GenerateToken(user.ID, "customer", jwtutils.RefreshToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("refresh generate token error")
	}

	// What: บันทึก session ลง DB พร้อม metadata (IP, device, expiry)
	// Why:  เก็บ session ไว้เพื่อ validate และ revoke token ได้ในภายหลัง
	session := &domain.Session{
		UserID:       &user.ID,
		AdminID:      nil,
		RefreshToken: refreshToken,
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		// Why: ExpiredAt ต้องตรงกับ JWT expiry ของ refresh token
		// TODO: ดึงค่า expiry จาก jwtService แทนการ hardcode 7 วัน เพื่อให้ sync กันเสมอ
		ExpiredAt: time.Now().Add(7 * 24 * time.Hour),
		IsRevoked: false,
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

// What: login flow สำหรับ admin — เหมือน LoginUser แต่ใช้ username แทน email
// Why:  admin เป็น internal account ไม่จำเป็นต้องมี email
func (s *authService) LoginAdmin(ctx context.Context, username, password, ipAddress, deviceInfo string) (*dto.LoginResponse, error) {
	// What: FindByUserName คืน nil (ไม่ใช่ error) ถ้าไม่เจอ — ต้องเช็ค nil แยก
	admin, err := s.adminRepo.FindByUsername(ctx, username)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	// Why: แยก "ไม่เจอ" กับ "DB error" ออกจากกัน เพื่อ response ที่ถูกต้อง
	if admin == nil {
		return nil, errs.NewUnauthorizedError("invalid credentials")
	}

	err = admin.CheckPassword(password)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnauthorizedError("incorrect password")
	}

	// What: ออก token ด้วย role "admin" เพื่อให้ middleware แยกสิทธิ์ได้
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

// What: revoke session ที่ตรงกับ refresh token ที่ส่งมา
// Why:  การ revoke ทำที่ DB เพราะ JWT ไม่มีวิธี invalidate ก่อนหมดอายุ
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	// What: ค้นหา session ก่อน เพื่อตรวจสอบสถานะปัจจุบัน
	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		logs.Error(err)
		return errs.NewNotFoundError("session not found")
	}

	// What: ป้องกัน double-logout — token ที่ revoke แล้วไม่ต้อง revoke ซ้ำ
	if session.IsRevoked {
		return errs.NewValidationError("token is already revoked")
	}

	// What: mark session ว่า revoked แล้ว (soft invalidation)
	if err := s.sessionRepo.RevokeSession(ctx, refreshToken); err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	return nil
}

// What: ใช้ refresh token ที่ valid เพื่อออก access token ใหม่
// Why:  access token มีอายุสั้น — user ไม่ต้อง login ใหม่ทุกครั้งที่หมดอายุ
// TODO: พิจารณา refresh token rotation (ออก refresh ใหม่ + revoke อันเก่า)
//       ช่วยลด risk กรณี refresh token ถูก leak
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dto.LoginResponse, error) {
	// What: validate signature และ claims ของ JWT ก่อน
	claims, err := s.jwtService.ValidateToken(refreshToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("invalid refresh token signature")
	}

	// Why: ป้องกัน panic ถ้า ValidateToken คืน nil claims (defensive check)
	if claims == nil {
		return nil, errs.NewValidationError("invalid token claims")
	}

	// What: ป้องกันการใช้ access token มา refresh (ต้องเป็น refresh token เท่านั้น)
	if claims.Type != jwtutils.RefreshToken {
		return nil, errs.NewNotFoundError("invalid token type")
	}

	// What: ตรวจสอบว่า session ยังอยู่ใน DB และยังไม่ถูก revoke
	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	// What: ถ้า session ถูก revoke แล้ว (logout ก่อนหน้า) ให้ปฏิเสธ
	if session.IsRevoked {
		return nil, errs.NewNotFoundError("token has been revoked")
	}

	// What: ตรวจสอบ expiry จาก DB (ไม่ใช่แค่ JWT) — ป้องกัน edge case
	if time.Now().After(session.ExpiredAt) {
		return nil, errs.NewValidationError("refresh token expired")
	}

	// What: หา subject ID จาก session record
	// Why:  ต้องดึงจาก DB ไม่ใช่ JWT เพราะต้องการ verify ว่า session ยังถูกต้อง
	var subjectID uint
	if session.UserID != nil {
		// What: session เป็นของ user
		subjectID = *session.UserID
	} else if session.AdminID != nil {
		// What: session เป็นของ admin
		subjectID = *session.AdminID
	} else {
		// What: ข้อมูล session ใน DB ผิดปกติ (ไม่มีทั้ง UserID และ AdminID)
		return nil, errs.NewUnexpectedError()
	}

	// What: ออก access token ใหม่ โดยคง role เดิมจาก claims
	newAccessToken, err := s.jwtService.GenerateToken(subjectID, claims.Role, jwtutils.AccessToken)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("access generate token error")
	}

	// What: คืน access token ใหม่ + refresh token เดิม (ไม่ rotate)
	return &dto.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken,
	}, nil
}
