package authsvc

import (
	"context"
	"testing"

	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const authTestSecretKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestLoginRejectsInvalidCSRFToken(t *testing.T) {
	service := New(nil, nil, jwt.TokenService{}, nil, authTestSecretKey)
	_, err := service.Login(context.Background(), authdto.LoginInput{Email: "admin@example.test", Password: "password", CSRFToken: "csrf"})
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("invalid CSRF error=%v", err)
	}
}

type loginFacts struct {
	repository.UserStore
	email string
}

func (f *loginFacts) UserByEmail(_ context.Context, email string) (model.User, error) {
	f.email = email
	return model.User{}, repository.ErrNotFound
}

func TestLoginDoesNotAcceptUsername(t *testing.T) {
	facts := &loginFacts{}
	service := New(facts, nil, jwt.TokenService{}, nil, authTestSecretKey)
	token, err := service.NewCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Login(context.Background(), authdto.LoginInput{Email: "admin", Password: "password", CSRFToken: token})
	if !apperror.IsKind(err, apperror.KindUnauthorized) || facts.email != "admin" {
		t.Fatalf("email=%q error=%v", facts.email, err)
	}
}

func TestChangePasswordRejectsWrongOldPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	service := New(nil, nil, jwt.TokenService{}, nil, authTestSecretKey)
	err = service.ChangePassword(context.Background(), authdto.ChangePasswordInput{User: model.User{PasswordHash: string(hash)}, OldPassword: "wrong", NewPassword: "newpass1"})
	if !apperror.IsKind(err, apperror.KindUnauthorized) {
		t.Fatalf("wrong old password error=%v", err)
	}
}
