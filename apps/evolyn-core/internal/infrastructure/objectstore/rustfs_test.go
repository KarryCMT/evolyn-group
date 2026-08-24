package objectstore

import "testing"

func TestParseEndpoint(t *testing.T) {
	cases := []struct {
		name          string
		raw           string
		defaultSecure bool
		wantEndpoint  string
		wantSecure    bool
		wantErr       bool
	}{
		{"host port", "127.0.0.1:9000", false, "127.0.0.1:9000", false, false},
		{"https public", "https://storage.example.com", false, "storage.example.com", true, false},
		{"http overrides ssl", "http://storage.example.com", true, "storage.example.com", false, false},
		{"path rejected", "https://storage.example.com/rustfs", false, "", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotEndpoint, gotSecure, err := parseEndpoint(tc.raw, tc.defaultSecure)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseEndpoint() error = %v", err)
			}
			if gotEndpoint != tc.wantEndpoint || gotSecure != tc.wantSecure {
				t.Fatalf("parseEndpoint() = (%q, %t), want (%q, %t)", gotEndpoint, gotSecure, tc.wantEndpoint, tc.wantSecure)
			}
		})
	}
}
