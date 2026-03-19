package service

import (
	"auth_service/internal/core/domain"
	repo "auth_service/internal/core/port/repo"
	service "auth_service/internal/core/port/service"
	dto "auth_service/internal/core/port/service/dto"
	"context"
	"errors"
	"errs"
	"logs"
)

// What: userService จัดการ business logic ที่เกี่ยวกับ user account
// Why:  แยกออกจาก authService เพราะ concern ต่างกัน —
//
//	authService = token/session, userService = profile/password
type userService struct {
	userRepo repo.UserRepository
}

// What: constructor — inject repository
func NewUserService(userRepo repo.UserRepository) service.UserService {
	return &userService{userRepo: userRepo}
}

// What: สร้าง user ใหม่ พร้อม validate ว่า email ไม่ซ้ำ
// Why:  ตรวจ email ซ้ำที่ service layer แทน DB constraint
//
//	เพื่อให้ error message ชัดเจน ไม่ต้อง parse DB error
func (s *userService) RegisterUser(ctx context.Context, newUserReq *dto.CreateUserRequest, newUserPassReq string) (*dto.UserResponse, error) {
	// What: map DTO → domain model (ยังไม่มี password)
	newUserDomain := &domain.User{
		FirstName: newUserReq.FirstName,
		LastName:  newUserReq.LastName,
		Email:     newUserReq.Email,
		Phone:     newUserReq.Phone,
		Address:   newUserReq.Address,
	}

	// What: hash password แล้วเก็บไว้ใน domain — ไม่เคย store plain-text
	if err := newUserDomain.SetPassword(newUserPassReq); err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("password must be at least 8 characters")
	}

	// What: ตรวจสอบว่า email นี้มีในระบบอยู่แล้วหรือไม่
	_, err := s.userRepo.FindByEmail(ctx, newUserDomain.Email)
	if err != nil {
		// Why: ถ้า error ไม่ใช่ ErrUserNotFound แปลว่า DB มีปัญหา ไม่ใช่ "ไม่เจอ"
		if !errors.Is(err, domain.ErrUserNotFound) {
			logs.Error(err)
			return nil, errs.NewUnexpectedError()
		}
		// What: err == ErrUserNotFound = email ยังไม่มี → ผ่านได้
	} else {
		// What: err == nil หมายความว่าหาเจอ → email ซ้ำ
		logs.Error(err)
		return nil, errs.NewConflictError("email already exists")
	}

	// What: บันทึก user ลง DB
	if err = s.userRepo.CreateUser(ctx, newUserDomain); err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	// What: คืน response ที่ตัด sensitive field (password) ออกแล้ว
	return dto.ToUserResponse(newUserDomain), nil
}

// What: อัปเดต profile ของ user (ชื่อ, เบอร์, ที่อยู่)
// Why:  ใช้ domain method UpdateUserProfile เพื่อ partial update — ป้องกัน overwrite field สำคัญ
// TODO: รองรับ email update โดยต้อง verify email ใหม่ก่อน
func (s *userService) UpdateUserProfile(ctx context.Context, userID uint, userUpdateReq *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	// What: สร้าง domain object จาก request — ใช้เป็น "patch" object
	userDomain := &domain.User{
		FirstName: userUpdateReq.FirstName,
		LastName:  userUpdateReq.LastName,
		Phone:     userUpdateReq.Phone,
		Address:   userUpdateReq.Address,
	}

	// What: ดึง user ปัจจุบันจาก DB ก่อน เพื่อ merge กับข้อมูลที่ส่งมา
	existingUser, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	if existingUser == nil {
		return nil, errs.NewNotFoundError("user not found")
	}

	// What: apply การเปลี่ยนแปลงบน domain object (เฉพาะ field ที่ไม่ว่าง)
	updatedUser := existingUser.UpdateUserProfile(userDomain)

	// What: บันทึก user ที่อัปเดตแล้วกลับไปยัง DB
	if err = s.userRepo.UpdateUser(ctx, updatedUser); err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	return dto.ToUserResponse(updatedUser), nil
}

// What: เปลี่ยน password — ต้องยืนยัน old password ก่อนเสมอ
// Why:  ป้องกัน attacker ที่ขโมย session ของคนอื่นแล้วเปลี่ยน password ได้ทันที
func (s *userService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	if user == nil {
		return errs.NewNotFoundError("user not found")
	}

	// What: domain method ตรวจสอบ old password และ hash new password ในครั้งเดียว
	if err := user.ChangePassword(oldPassword, newPassword); err != nil {
		logs.Error(err)
		// Why: แยก "รหัสผ่านเก่าผิด" กับ "validation error" เพื่อ HTTP status code ที่ถูกต้อง
		if errors.Is(err, domain.ErrIncorrectPassword) {
			return errs.NewUnauthorizedError("incorrect old password")
		}
		return errs.NewValidationError("password change failed")
	}

	// What: save user กลับไป — password hash ถูก update แล้วใน domain object
	if err = s.userRepo.UpdateUser(ctx, user); err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	return nil
}

// What: ดึงโปรไฟล์ user และ map เป็น DTO ก่อนคืนให้ handler
// Why:  ตัด sensitive field (เช่น password hash) ออก ไม่ expose ไปยัง HTTP response
func (s *userService) GetProfile(ctx context.Context, id uint) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	if user == nil {
		return nil, errs.NewNotFoundError("user not found")
	}

	return dto.ToUserResponse(user), nil
}
