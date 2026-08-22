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
		{name: "logical directory", sourceType: "directory", source: "./models/bge-m3", valid: true},
		{name: "logical directory without compose prefix", sourceType: "directory", source: "models/bge-m3"},
		{name: "host directory", sourceType: "directory", source: "D:/var/lib/pomelo-models/bge-m3", hostPath: true, valid: true},
		{name: "absolute logical directory", sourceType: "directory", source: "D:/var/lib/pomelo-models/bge-m3", valid: true},
		{name: "absolute controlled file", sourceType: "controlled_file", source: "/etc/pomelo/app.env", valid: true},
		{name: "relative controlled file", sourceType: "controlled_file", source: "./config/app.env", valid: true},
		{name: "bare controlled file", sourceType: "controlled_file", source: "config/app.env"},
		{name: "drive-relative directory", sourceType: "directory", source: "D:var/lib/pomelo-models"},
		{name: "backslash Windows directory", sourceType: "directory", source: `D:\var\lib\pomelo-models`},
		{name: "UNC directory", sourceType: "directory", source: `\\server\share\models`},
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
