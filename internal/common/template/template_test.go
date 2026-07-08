package templatex

import (
	"strings"
	"testing"
)

func TestRenderLiquidTemplate(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		values map[string]any
		want   string
	}{
		{
			name:  "simple variables",
			input: "image: {{ app.code }}.{{ config.domain_suffix }}",
			values: map[string]any{
				"app":    map[string]any{"code": "demo"},
				"config": map[string]any{"domain_suffix": "lvh.me"},
			},
			want: "image: demo.lvh.me",
		},
		{
			name:   "default filter",
			input:  "{{ IMAGE | default: 'nginx' }}",
			values: map[string]any{"IMAGE": ""},
			want:   "nginx",
		},
		{
			name:   "number value",
			input:  "port={{ PORT }}",
			values: map[string]any{"PORT": 8080},
			want:   "port=8080",
		},
		{
			name:   "boolean value",
			input:  "enabled={{ ENABLED }}",
			values: map[string]any{"ENABLED": true},
			want:   "enabled=true",
		},
		{
			name:   "if expression",
			input:  "{% if ENABLED %}on{% else %}off{% endif %}",
			values: map[string]any{"ENABLED": true},
			want:   "on",
		},
		{
			name:   "elsif expression",
			input:  "{% if MODE == 'on' %}on{% elsif MODE == 'auto' %}auto{% else %}off{% endif %}",
			values: map[string]any{"MODE": "auto"},
			want:   "auto",
		},
		{
			name:   "for expression",
			input:  "{% for item in items %}{{ item }}{% endfor %}",
			values: map[string]any{"items": []string{"a", "b"}},
			want:   "ab",
		},
		{
			name:   "upper filter",
			input:  "{{ name | upcase }}",
			values: map[string]any{"name": "demo"},
			want:   "DEMO",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Render(tc.input, tc.values)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestRenderDefaultFilterAllowsMissingVariable(t *testing.T) {
	got, err := Render("{{ IMAGE | default: 'nginx' }}", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "nginx" {
		t.Fatalf("expected %q, got %q", "nginx", got)
	}
}

func TestRenderMissingVariableReturnsError(t *testing.T) {
	_, err := Render("{{ MISSING }}", map[string]any{})
	if err == nil {
		t.Fatal("expected missing variable error")
	}
}

func TestRenderMissingDottedVariableReturnsError(t *testing.T) {
	_, err := Render("{{ app.missing }}", map[string]any{"app": map[string]any{"code": "demo"}})
	if err == nil {
		t.Fatal("expected missing variable error")
	}
}

func TestRenderMissingVariableInConditionReturnsError(t *testing.T) {
	_, err := Render("{% if ENABLED %}on{% endif %}", map[string]any{})
	if err == nil {
		t.Fatal("expected missing variable error")
	}
}

func TestRenderMissingLoopCollectionReturnsError(t *testing.T) {
	_, err := Render("{% for item in items %}{{ item }}{% endfor %}", map[string]any{})
	if err == nil {
		t.Fatal("expected missing variable error")
	}
}

func TestRenderSyntaxErrorReturnsError(t *testing.T) {
	_, err := Render("{% if ENABLED %}", map[string]any{"ENABLED": true})
	if err == nil {
		t.Fatal("expected syntax error")
	}
}

func TestRenderUnknownFilterReturnsError(t *testing.T) {
	_, err := Render("{{ name | does_not_exist }}", map[string]any{"name": "demo"})
	if err == nil {
		t.Fatal("expected unknown filter error")
	}
	if strings.Contains(err.Error(), "nginx") {
		t.Fatalf("unexpected error: %v", err)
	}
}
