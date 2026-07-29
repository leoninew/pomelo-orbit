package commandline

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "shell command",
			input: `sh -c 'exec redis-server --requirepass "$${REDIS_PASSWORD}" --maxmemory 128mb --maxmemory-policy allkeys-lru'`,
			want:  []string{"sh", "-c", `exec redis-server --requirepass "$${REDIS_PASSWORD}" --maxmemory 128mb --maxmemory-policy allkeys-lru`},
		},
		{name: "escaped space", input: `server --label=hello\ world`, want: []string{"server", "--label=hello world"}},
		{name: "empty", input: "", want: []string{}},
		{name: "does not expand variables", input: "echo $HOME", want: []string{"echo", "$HOME"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse(test.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Parse() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseRejectsUnclosedQuote(t *testing.T) {
	if _, err := Parse("redis-server '--requirepass"); err == nil {
		t.Fatal("Parse() error = nil, want an error")
	}
}

func TestFormatRoundTrip(t *testing.T) {
	want := []string{"sh", "-c", `exec redis-server --requirepass "$${REDIS_PASSWORD}"`, "", "it's-safe"}
	got, err := Parse(Format(want))
	if err != nil {
		t.Fatalf("Parse(Format()) error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse(Format()) = %#v, want %#v", got, want)
	}
}
