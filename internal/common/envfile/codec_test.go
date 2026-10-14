package envfile

import (
	"reflect"
	"testing"
)

func TestCodecPreservesLiteralValues(t *testing.T) {
	values := map[string]string{
		"EMPTY": "", "ZERO": "0012", "BOOL": "false", "DOLLAR": "${SECRET} $HOME",
		"QUOTES": "'quoted' and \"double\"", "PATH": " C:\\folder\\ ", "TRAILING": "C:\\folder\\",
		"MULTILINE": "line one\nline two\r\n", "COMMENT": "value # literal",
		"ESCAPES": "\x01\t\b\f", "UNICODE": "\u4e2d\u6587 \U0001f600\u2028",
	}
	content, err := Encode(values)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(content)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(values, decoded) {
		t.Fatalf("round-trip changed values: %#v", decoded)
	}
	decoded, err = Decode([]byte("# comment\nexport KEY=\"value\" # comment\nLITERAL='${HOME}'\nRAW=${HOME}\n"))
	if err != nil || decoded["KEY"] != "value" || decoded["LITERAL"] != "${HOME}" || decoded["RAW"] != "${HOME}" {
		t.Fatalf("decode = %#v, %v", decoded, err)
	}
}
