package workspacepath

import "testing"

func TestServiceDirectoriesUseTargetPlatform(t *testing.T) {
	for _, tc := range []struct{ value, platform, want string }{
		{"/srv/service with spaces", "linux", "/srv/service with spaces"},
		{"~/custom/api", "linux", "~/custom/api"},
		{"D:\\Service Data\\api", "windows", "D:/Service Data/api"},
		{"D:/Service Data/api/", "windows", "D:/Service Data/api"},
	} {
		actual, err := NormalizeServiceDirectory(tc.value, tc.platform)
		if err != nil || actual != tc.want {
			t.Fatalf("%q (%s): got %q, %v", tc.value, tc.platform, actual, err)
		}
	}
	for _, value := range []string{"relative/api", "/srv/../other", "/", ""} {
		if _, err := NormalizeServiceDirectory(value, "linux"); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	for _, value := range []string{"D:/", `D:\`, "D:/./"} {
		if _, err := NormalizeServiceDirectory(value, "windows"); err == nil {
			t.Fatalf("accepted Windows drive root %q", value)
		}
	}
	if !OverlappingDirectories("D:/Services/API", "d:/services/api/data", "windows") {
		t.Fatal("Windows overlap is case insensitive")
	}
	if OverlappingDirectories("/srv/api", "/srv/api-two", "linux") {
		t.Fatal("distinct siblings overlap")
	}
}
