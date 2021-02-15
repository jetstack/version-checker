package semver

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := map[string]struct {
		input       string
		expVersion  [3]int64
		expMetadata string
	}{
		"No input should no output": {
			"",
			[3]int64{0, 0, 0},
			"",
		},
		"No numbers should no output": {
			"v",
			[3]int64{0, 0, 0},
			"v",
		},
		"Not matching semver should no output": {
			"hello-1.2.3",
			[3]int64{0, 0, 0},
			"hello-1.2.3",
		},
		"1 -> [1 0 0]": {
			"1",
			[3]int64{1, 0, 0},
			"",
		},
		"1.2 -> [1 2 0]": {
			"1.2",
			[3]int64{1, 2, 0},
			"",
		},
		"1.0.1 -> [1 0 1]": {
			"1.0.1",
			[3]int64{1, 0, 1},
			"",
		},
		"v1.0.1 -> [1 0 1]": {
			"v1.0.1",
			[3]int64{1, 0, 1},
			"",
		},
		"v1.0.1-debian-3.hello-world-12 -> [1 0 1]": {
			"v1.0.1-debian-3.hello-world-12",
			[3]int64{1, 0, 1},
			"-debian-3.hello-world-12",
		},
		"v1.0.1- -> [1 0 1]": {
			"v1.0.1-",
			[3]int64{1, 0, 1},
			"-",
		},
		"v1.2-alpha -> [1 2 3]": {
			"v1.0.1-",
			[3]int64{1, 0, 1},
			"-",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s := Parse(test.input)
			if !reflect.DeepEqual(s.version, test.expVersion) {
				t.Errorf("got unexpected version output, exp=%v got=%v",
					test.expVersion, s.version)
			}

			if test.expMetadata != s.metadata {
				t.Errorf("unexpected metadata, exp=%s got=%s",
					test.expMetadata, s.metadata)
			}
		})
	}
}

func TestMajorMinorPatch(t *testing.T) {
	tests := map[string]struct {
		input   string
		expInts [3]int64
	}{
		"No input should no input": {
			"",
			[3]int64{0, 0, 0},
		},
		"1 -> [1]": {
			"1",
			[3]int64{1, 0, 0},
		},
		"1.0.1 -> [1 0 1]": {
			"1.0.1",
			[3]int64{1, 0, 1},
		},
		"v1.0.1 -> [1 0 1]": {
			"v1.0.1",
			[3]int64{1, 0, 1},
		},
		"v1.3.hello1-debian-3.hello-world-12 -> [1 3 1 3 12]": {
			"v1.3.hello1-debian-3.hello-world-12",
			[3]int64{1, 3, 0},
		},
	}

	testPart := func(name string, exp int64, f func() int64) {
		if got := f(); exp != got {
			t.Errorf("unexpected %s int, exp=%v got=%v",
				name, exp, got)
		}
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			s := Parse(test.input)
			testPart("major", test.expInts[0], s.Major)
			testPart("minor", test.expInts[1], s.Minor)
			testPart("patch", test.expInts[2], s.Patch)
		})
	}
}

func TestLessThan(t *testing.T) {
	tests := map[string]struct {
		first, second string
		lessThan      bool
	}{
		"No input should be false": {
			"", "",
			false,
		},
		"Arbitrary strings should be false": {
			"hello", "aworld",
			false,
		},
		"'' is less than 1": {
			"", "1",
			true,
		},
		"1 is not less than ''": {
			"1", "",
			false,
		},
		"If first doesn't contain number, second does, true": {
			"hello", "world-1",
			true,
		},
		"If first contains number, second doesn't, but not less than false": {
			"hello-1", "aworld",
			false,
		},
		"If first same, false": {
			"v0.1.2", "v0.1.2",
			false,
		},
		"If first less, true": {
			"v0.1.2", "v0.1.3",
			true,
		},
		"If second alpha, false": {
			"v0.1.2", "v0.1.3-alpha",
			false,
		},
		"If second alpha with num, false": {
			"v0.1.2", "v0.1.3-alpha.0",
			false,
		},
		"If first older alpha, true": {
			"v0.1.3-alpha.0", "v0.1.3-alpha.1",
			true,
		},
		"If first beta, false": {
			"v0.1.3-beta.0", "v0.1.3-alpha.1",
			false,
		},
		"If first alpha, second beta true": {
			"v0.1.3-12.alpha.1", "v0.1.3-12.beta.1",
			true,
		},
		"If first alpha smaller number, true": {
			"v0.1.3-alpha.103.gke", "v0.1.3-alpha.0113.gke",
			true,
		},
		"If first less with complications, true": {
			"v1.3.hello1-debian-3.hello-world-12", "v1.3.hello1-debian-9.hello-world-12",
			true,
		},
		"If first more with complications, false": {
			"v1.3.hello1-debian-9.hello-world-125", "v1.3.hello1-debian-9.hello-world-115",
			false,
		},
		"If same with complications, false": {
			"v1.3.hello1-debian-9.hello-world-12", "v1.3.hello1-debian-9.hello-world-12",
			false,
		},
		"If same with complications for rnumber": {
			"0.21.0-debian-10-r9", "0.21.0-debian-10-r39",
			true,
		},

		"If same with complications for rnumber extra": {
			"0.21.0-debian-10-r9-hello", "0.21.0-debian-10-r39-hello",
			true,
		},
		"If same with complications for rnumber extra reverse": {
			"0.21.0-debian-10-r39-hello", "0.21.0-debian-10-r9-hello",
			false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if Parse(test.first).LessThan(Parse(test.second)) != test.lessThan {
				t.Errorf("unexpected less than, first=%s second=%s expLessThan=%t",
					test.first, test.second, test.lessThan)
			}
		})
	}
}
