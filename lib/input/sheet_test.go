package input

import (
	"testing"
)

func TestNew(t *testing.T) {
	filePath := "20220101-12345678-TBD-K.xlsx"
	actual := New(filePath)
	expected := Sheet{
		Config: Config{true, true},
		Header: Header{
			FileName:  "pncheck_" + filePath,
			OrderType: 購入,
		},
	}
	if actual.Config != expected.Config {
		t.Errorf("got %#v, want: %#v", actual.Config, &expected.Config)
	}
	if actual.Header != expected.Header {
		t.Errorf("got %#v, want: %#v", actual.Header, &expected.Header)

	}
}
