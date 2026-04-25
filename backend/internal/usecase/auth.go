package usecase

import (
	"context"
	"errors"
	"time"

	"cbt-enterprise/internal/domain"
	"cbt-enterprise/pkg/jwt"
	"cbt-enterprise/pkg/bcrypt"

	"github.com/google/uuid"
)

type AuthUsecase struct {
	userRepo    domain.UserRepository
	pesertaRepo domain.PesertaRepository
	guruRepo    domain.GuruRepository
	jwtPkg      *jwt.JWT
}

func NewAuthUsecase(ur domain.UserRepository, pr domain.PesertaRepository, gr domain.GuruRepository, j *jwt.JWT) *AuthUsecase {
	return &AuthUsecase{userRepo: ur, pesertaRepo: pr, guruRepo: gr, jwtPkg: j}
}

type RegisterInput struct {
	Nama     string
	Email    string
	Password string
	Role     domain.Role
	// Optional
	NIS   string
	Kelas string
	NIP   string
	Mapel string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthOutput struct {
	Token string
	User  *domain.User
}

func (uc *AuthUsecase) Register(ctx context.Context, input RegisterInput) (*AuthOutput, error) {
	existing, _ := uc.userRepo.FindByEmail(ctx, input.Email)
	if existing != nil {
		return nil, errors.New("email sudah terdaftar")
	}

	hashed, err := bcrypt.Hash(input.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:       uuid.New(),
		Nama:     input.Nama,
		Email:    input.Email,
		Password: hashed,
		Role:     input.Role,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	switch input.Role {
	case domain.RolePeserta:
		p := &domain.Peserta{
			ID:     uuid.New(),
			UserID: user.ID,
			NIS:    input.NIS,
			Kelas:  input.Kelas,
		}
		if err := uc.pesertaRepo.Create(ctx, p); err != nil {
			return nil, err
		}
	case domain.RoleGuru:
		g := &domain.Guru{
			ID:     uuid.New(),
			UserID: user.ID,
			NIP:    input.NIP,
			Mapel:  input.Mapel,
		}
		if err := uc.guruRepo.Create(ctx, g); err != nil {
			return nil, err
		}
	}

	token, err := uc.jwtPkg.Generate(user.ID.String(), string(user.Role), time.Hour*24)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{Token: token, User: user}, nil
}

func (uc *AuthUsecase) Login(ctx context.Context, input LoginInput) (*AuthOutput, error) {
	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, errors.New("email atau password salah")
	}

	if !bcrypt.Check(input.Password, user.Password) {
		return nil, errors.New("email atau password salah")
	}

	token, err := uc.jwtPkg.Generate(user.ID.String(), string(user.Role), time.Hour*24)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{Token: token, User: user}, nil
}
