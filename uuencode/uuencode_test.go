package uuencode_test

import (
	"testing"

	"github.com/NEU-SNS/ReverseTraceroute/uuencode"
)

func TestEncode(t *testing.T) {
	for _, test := range []struct {
		raw string
		enc string
	}{
		{raw: "this is a test string", enc: "5=&AI<R!I<R!A('1E<W0@<W1R:6YG\n`\n"},
		{raw: "short test", enc: "*<VAO<G0@=&5S=```\n`\n"},
		{raw: "This is a much longer test which should make for multiple lines",
			enc: "M5&AI<R!I<R!A(&UU8V@@;&]N9V5R('1E<W0@=VAI8V@@<VAO=6QD(&UA:V4@\n29F]R(&UU;'1I<&QE(&QI;F5S\n`\n"},
	} {
		enc, err := uuencode.UUEncode([]byte(test.raw))
		if err != nil {
			t.Fatalf("Error encoding. Got[%v], expected[<nil>]", err)
		}
		if string(enc) != test.enc {
			t.Fatalf("Error encoding. Got[%s], expected[%s]", string(enc), test.enc)
		}
	}
}

func TestDecode(t *testing.T) {
	for _, test := range []struct {
		dec string
		enc string
	}{
		{dec: "this is a test string", enc: "5=&AI<R!I<R!A('1E<W0@<W1R:6YG\n`\n"},
		{dec: "short test", enc: "*<VAO<G0@=&5S=```\n`\n"},
		{dec: "This is a much longer test which should make for multiple lines",
			enc: "M5&AI<R!I<R!A(&UU8V@@;&]N9V5R('1E<W0@=VAI8V@@<VAO=6QD(&UA:V4@\n29F]R(&UU;'1I<&QE(&QI;F5S\n`\n"},
	} {
		dec, err := uuencode.UUDecode([]byte(test.enc))
		if err != nil {
			t.Fatalf("Error decoding. Got[%v], expected[<nil>]", err)
		}
		if string(dec) != test.dec {
			t.Fatalf("Error encoding. Got[%s], expected[%s]", string(dec), test.dec)
		}
	}
}
