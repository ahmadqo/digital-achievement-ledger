package service

import (
	"context"
	"errors"

	"github.com/ahmadqo/digital-achievement-ledger/internal/config"
	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
	"github.com/ahmadqo/digital-achievement-ledger/internal/repository"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Request & Response DTOs
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User  model.UserResponse `json:"user"`
	Token utils.TokenPair    `json:"token"`
}

type RegisterRequest struct {
	Name     string     `json:"name"`
	Email    string     `json:"email"`
	Password string     `json:"password"`
	Role     model.Role `json:"role"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Errors
var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrAccountDisabled    = errors.New("akun tidak aktif, hubungi administrator")
	ErrEmailAlreadyExists = errors.New("email sudah terdaftar")
)

type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Register(ctx context.Context, req RegisterRequest) (*model.UserResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error)
	Me(ctx context.Context, userID string) (*model.UserResponse, error)
}

type authService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *authService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	// Cari user by email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Cek status aktif
	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	// Validasi password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate token
	claims := model.JWTClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   string(user.Role),
		Name:   user.Name,
	}

	tokenPair, err := utils.GenerateTokenPair(
		claims,
		s.cfg.JWT.Secret,
		s.cfg.JWT.ExpireHours,
		s.cfg.JWT.RefreshExpHours,
	)
	if err != nil {
		return nil, err
	}

	userResp := user.ToResponse()
	return &LoginResponse{
		User:  userResp,
		Token: *tokenPair,
	}, nil
}

func (s *authService) Register(ctx context.Context, req RegisterRequest) (*model.UserResponse, error) {
	// Cek email sudah ada
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Default role jika kosong
	if req.Role == "" {
		req.Role = model.RoleOperator
	}

	user := &model.User{
		ID:       uuid.New(),
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
		Role:     req.Role,
		IsActive: true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	resp := user.ToResponse()
	return &resp, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error) {
	// Validasi refresh token
	claims, err := utils.ValidateToken(refreshToken, s.cfg.JWT.Secret)
	if err != nil {
		return nil, errors.New("refresh token tidak valid atau sudah expired")
	}

	// Pastikan user masih ada dan aktif
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, errors.New("token tidak valid")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil || !user.IsActive {
		return nil, errors.New("akun tidak ditemukan atau tidak aktif")
	}

	// Generate token baru
	newClaims := model.JWTClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   string(user.Role),
		Name:   user.Name,
	}

	return utils.GenerateTokenPair(
		newClaims,
		s.cfg.JWT.Secret,
		s.cfg.JWT.ExpireHours,
		s.cfg.JWT.RefreshExpHours,
	)
}

func (s *authService) Me(ctx context.Context, userID string) (*model.UserResponse, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("user ID tidak valid")
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user tidak ditemukan")
	}

	resp := user.ToResponse()
	return &resp, nil
}