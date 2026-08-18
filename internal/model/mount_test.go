package model

import "testing"

func TestValidateMountSource(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
		source     string
		hostPath   bool
		valid      bool
	}{
		{name: "logical directory", sourceType: "directory", source: "models/bge-m3", valid: true},
		{name: "host directory", sourceType: "directory", source: "D:/var/lib/pomelo-models/bge-m3", hostPath: true, valid: true},
		{name: "absolute logical directory", sourceType: "directory", source: "D:/var/lib/pomelo-models/bge-m3"},
		{name: "relative host directory", sourceType: "directory", source: "models/bge-m3", hostPath: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateMountSource(test.sourceType, test.source, test.hostPath)
			if test.valid && err != nil {
				t.Fatalf("ValidateMountSource() error = %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("ValidateMountSource() error = nil")
			}
		})
	}
}
