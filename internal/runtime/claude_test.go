package runtime

import "testing"

func TestResolveClaudePackage(t *testing.T) {
	cases := []struct {
		raw         string
		wantPackage claudePackage
		wantVersion string
	}{
		{"", packageCC, ""},
		{"  ", packageCC, ""},
		{"cc", packageCC, ""},
		{"cc:1.2.3", packageCC, "1.2.3"},
		{"ccb", packageCCB, ""},
		{"ccb:0.4.0", packageCCB, "0.4.0"},
		{"1.2.3", packageCC, "1.2.3"},
		{"  2.0.0-beta ", packageCC, "2.0.0-beta"},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			gotPkg, gotVer := ResolveClaudePackage(tc.raw)
			if gotPkg != tc.wantPackage {
				t.Fatalf("package = %+v want %+v", gotPkg, tc.wantPackage)
			}
			if gotVer != tc.wantVersion {
				t.Fatalf("version = %q want %q", gotVer, tc.wantVersion)
			}
		})
	}
}
