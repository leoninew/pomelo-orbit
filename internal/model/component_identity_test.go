package model

import "testing"

func TestComponentIdentity(t *testing.T) {
	for _, user := range []string{"", "1000", "1000:1000", "orbit", "orbit:docker"} {
		if err := ValidateComponentIdentity(&user, []string{"988", "docker"}); err != nil {
			t.Fatalf("user %q: %v", user, err)
		}
	}
	for _, user := range []string{"1000:", ":docker", "1000:1000:988", "orbit user", "orbit\x00"} {
		if err := ValidateComponentIdentity(&user, nil); err == nil {
			t.Fatalf("accepted invalid user %q", user)
		}
	}
	if err := ValidateComponentIdentity(nil, []string{""}); err == nil {
		t.Fatal("accepted an empty supplementary group")
	}
}
