package projectsvc

import (
	"testing"

	projectdto "github.com/leoninew/pomelo-orbit/internal/application/project/dto"
)

func TestProjectCodeValidation(t *testing.T) {
	for _, test := range []struct {
		code  string
		valid bool
	}{
		{code: "a", valid: true},
		{code: "a+b.c-1", valid: true},
		{code: "orbit.preflite.cn", valid: true},
		{code: "abcdefghijklmnop", valid: true},
		{code: "abcdefghijklmnopqrstuvwxyz012345", valid: true},
		{code: "1project"},
		{code: "project_name"},
		{code: "abcdefghijklmnopqrstuvwxyz0123456"},
	} {
		t.Run(test.code, func(t *testing.T) {
			_, _, err := normalizeAndValidateCreate(projectdto.CreateInput{Name: "Project", Code: test.code})
			if (err == nil) != test.valid {
				t.Fatalf("code %q valid = %t, want %t (error: %v)", test.code, err == nil, test.valid, err)
			}
		})
	}
}
