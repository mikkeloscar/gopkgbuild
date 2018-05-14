package pkgbuild

import "testing"

// Test version comparison
func TestVersionComparison(t *testing.T) {
	alphaNumeric := []Version{
		"1.0.1",
		"1.0.a",
		"1.0",
		"1.0rc",
		"1.0pre",
		"1.0p",
		"1.0beta",
		"1.0b",
		"1.0a",
	}
	numeric := []Version{
		"20141130",
		"012",
		"11",
		"3.0.0",
		"2.011",
		"2.03",
		"2.0",
		"1.2",
		"1.1.1",
		"1.1",
		"1.0.1",
		"1.0.0.0.0.0",
		"1.0",
		"1",
	}
	git := []Version{
		"r1000.b481c3c",
		"r37.e481c3c",
		"r36.f481c3c",
	}

	bigger := func(list []Version) {
		for i, v := range list {
			for _, v2 := range list[i:] {
				if v != v2 && !v.bigger(v2) {
					t.Errorf("%s should be bigger than %s", v, v2)
				}
			}
		}
	}

	smaller := func(list []Version) {
		for i := len(list) - 1; i >= 0; i-- {
			v := list[i]
			for _, v2 := range list[:i] {
				if v != v2 && v.bigger(v2) {
					t.Errorf("%s should be smaller than %s", v, v2)
				}
			}
		}
	}

	bigger(alphaNumeric)
	smaller(alphaNumeric)
	bigger(numeric)
	smaller(numeric)
	bigger(git)
	smaller(git)
}

// Test alphaCompare function
func TestAlphaCompare(t *testing.T) {
	if alphaCompare([]rune("test"), []rune("test")) != 0 {
		t.Error("should be 0")
	}

	if alphaCompare([]rune("test"), []rune("test123")) > 0 {
		t.Error("should be less than 0")
	}

	if alphaCompare([]rune("test123"), []rune("test")) < 0 {
		t.Error("should be greater than 0")
	}
}

// Test CompleteVersion comparisons
func TestCompleteVersionComparison(t *testing.T) {
	a := &CompleteVersion{
		Version: "2",
		Epoch:   1,
		Pkgrel:  Version("2"),
	}

	older := []string{
		"0:3-4",
		"1:2-1",
		"1:2-1.5",
		"1:1-1",
	}

	for _, o := range older {
		if b, err := NewCompleteVersion(o); err != nil {
			t.Errorf("%s fails to parse %v", o, err)
		} else if a.Older(b) || !a.Newer(b) {
			t.Errorf("%s should be older than %s", o, a.String())
		}
	}

	newer := []string{
		"2:1-1",
		"1:3-1",
		"1:2-3",
		"1:2-2.1",
	}

	for _, n := range newer {
		if b, err := NewCompleteVersion(n); err != nil {
			t.Errorf("%s fails to parse %v", n, err)
		} else if a.Newer(b) || !a.Older(b) {
			t.Errorf("%s should be newer than %s", n, a.String())
		}
	}

	equal := []string{
		"1:2-2",
		"1:2",
	}

	for _, n := range equal {
		if b, err := NewCompleteVersion(n); err != nil {
			t.Errorf("%s fails to parse %v", n, err)
		} else if a.Newer(b) || a.Older(b) || !a.Equal(b) {
			t.Errorf("%s should be equal to %s", n, a.String())
		}
	}

}

func TestCompleteVersionString(t *testing.T) {
	str := "42:3.14-1"
	version, _ := NewCompleteVersion(str)
	if version.String() != str {
		t.Errorf("%v should equal %s", version, str)
	}
}

func TestSatisfies(t *testing.T) {
	deps, _ := ParseDeps([]string{
		"a>1", "a<2",
		"b>=1", "b<=2",
		"c>=1:1", "c<=1:2",
		"d=1.2.3",
		"e=1.2.3-1",
	})

	versionPass := [][]string{
		{"1.1", "1.9", "1.99", "1.001", "0:1.1"},
		{"1.1", "1.9", "1.99", "1.001", "1", "2"},
		{"1:1.1", "1:1.9", "1:1.99", "1:1.001", "1:1", "1:2"},
		{"1.2.3", "1.2.3-1", "1.2.3-999", "0:1.2.3", "0:1.2.3-1", "0:1.2.3-999"},
		{"1.2.3-1", "0:1.2.3-1"},
	}

	versionFail := [][]string{
		{"0", "0.99", "2.001", "2"},
		{"0", "0.99", "2.001"},
		{"0:1.1", "2:1.9", "0:1.99", "2:1.001", "0:1", "2:2", "0:1.0", "2:2.0"},
		{"1:1.2.3", "1.2.3.0", "1.2.3.1"},
		{"1:1.2.3-1", "0:1.2.3-2", "1.2.3-3", "1.2.3-4"},
	}

	for i, dep := range deps {
		versions := versionPass[i]
		for _, versionStr := range versions {
			version, err := NewCompleteVersion(versionStr)
			if err != nil {
				t.Errorf("%s fails to parse %v", versionStr, err)
			}
			if !version.Satisfies(dep) {
				t.Errorf("%s should satisfy %+v", version, dep)
			}
		}
	}

	for i, dep := range deps {
		versions := versionFail[i]
		for _, versionStr := range versions {
			version, err := NewCompleteVersion(versionStr)
			if err != nil {
				t.Errorf("%s fails to parse %v", versionStr, err)
			}
			if version.Satisfies(dep) {
				t.Errorf("%s should not satisfy %+v", version, dep)
			}
		}
	}

}

// Benchmark rpmvercmp
func BenchmarkVersionCompare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rpmvercmp("1.0", "1.0.0")
	}
}

func TestUnicode(t *testing.T) {
	str := "13:2.0.0.α.r29.g18fc492-1"
	expected := CompleteVersion{
		Epoch:   13,
		Version: "2.0.0.α.r29.g18fc492",
		Pkgrel:  "1",
	}
	version, err := NewCompleteVersion(str)
	if err != nil {
		t.Error(err)
	} else if *version != expected {
		t.Errorf("%v should be %v", version, expected)
	}
}
