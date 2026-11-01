package routes

import "testing"

func TestImmutableAssetPathPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{path: "js/runtime-1.2.3.deadbeef.js", want: true},
		{path: "/js/runtime-1.2.3.deadbeef.js", want: true},
		{path: "/js/runtime-1.2.3.deadbeef.min.js", want: true},
		{path: "/js/runtime.js", want: false},
		{path: "/css/site.deadbeef.css", want: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			t.Parallel()
			if got := immutableAssetPathPattern.MatchString(test.path); got != test.want {
				t.Fatalf("immutableAssetPathPattern.MatchString(%q) = %t, want %t", test.path, got, test.want)
			}
		})
	}
}
