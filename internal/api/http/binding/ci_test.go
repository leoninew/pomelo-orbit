package binding

import (
	"strings"
	"testing"
)

func TestRepositoryWebhookUpdateReqBranchFilterPresence(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		wantBranchSet   bool
		wantBranchValue *string
	}{
		{
			name:          "omitted",
			body:          `{}`,
			wantBranchSet: false,
		},
		{
			name:            "set",
			body:            `{"branch_filter":"main"}`,
			wantBranchSet:   true,
			wantBranchValue: stringPtr("main"),
		},
		{
			name:          "explicit null clears value",
			body:          `{"branch_filter":null}`,
			wantBranchSet: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var req RepositoryWebhookUpdateReq
			if err := DecodeJSONReader(strings.NewReader(test.body), &req); err != nil {
				t.Fatal(err)
			}
			if req.BranchSet != test.wantBranchSet {
				t.Fatalf("BranchSet = %v, want %v", req.BranchSet, test.wantBranchSet)
			}
			if test.wantBranchValue == nil {
				if req.Body.BranchFilter != nil {
					t.Fatalf("BranchFilter = %q, want nil", *req.Body.BranchFilter)
				}
				return
			}
			if req.Body.BranchFilter == nil || *req.Body.BranchFilter != *test.wantBranchValue {
				t.Fatalf("BranchFilter = %v, want %q", req.Body.BranchFilter, *test.wantBranchValue)
			}
		})
	}
}

func TestRepositoryWebhookUpdateReqRejectsInvalidOrUnknownJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `{`},
		{name: "unknown field", body: `{"unknown":"value"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var req RepositoryWebhookUpdateReq
			if err := DecodeJSONReader(strings.NewReader(test.body), &req); err == nil {
				t.Fatal("expected decode error")
			}
		})
	}
}

func stringPtr(value string) *string {
	return &value
}
