package usersvc

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	userdto "github.com/leoninew/pomelo-orbit/internal/application/user/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestUserServiceValidatesFieldsBeforeSaving(t *testing.T) {
	for _, test := range []struct {
		name     string
		password string
		email    string
		valid    bool
	}{
		{name: "36 character password", password: strings.Repeat("a", 36), email: "operator@example.test", valid: true},
		{name: "37 character password", password: strings.Repeat("a", 37), email: "operator@example.test"},
		{name: "missing email", password: "secret1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &userCreateStore{}
			service := New(store, nil, nil)
			user, err := service.Create(context.Background(), userdto.CreateInput{Username: "operator", Password: test.password, Email: test.email})
			if !test.valid {
				if !apperror.IsKind(err, apperror.KindValidation) || len(store.saved) != 0 {
					t.Fatalf("error=%v saved=%d", err, len(store.saved))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(store.saved) != 1 || store.saved[0].Id != user.Id {
				t.Fatalf("saved users=%+v", store.saved)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(store.saved[0].PasswordHash), []byte(test.password)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type userCreateStore struct {
	repository.UserStore
	saved []model.User
}

func (*userCreateStore) UserByUsername(context.Context, string) (model.User, error) {
	return model.User{}, repository.ErrNotFound
}
func (*userCreateStore) UserByEmail(context.Context, string) (model.User, error) {
	return model.User{}, repository.ErrNotFound
}
func (s *userCreateStore) CreateUser(_ context.Context, user model.User) error {
	s.saved = append(s.saved, user)
	return nil
}
