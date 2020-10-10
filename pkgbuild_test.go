package pkgbuild

import "testing"

// Test version parsing
func TestVersionParsing(t *testing.T) {
	versions := map[string]bool{
		"1.0beta":   true,
		"1.0.0.0.2": true,
		"a.3_4":     true,
		"A.2":       true,
		"a~b~c":     true,
		"a.2Ø":      true,
		"2.0.0.α":   true,
		"_1.2":      false,
		".2":        false,
		"1.?":       false,
		"1.-":       false,
		"1 2":       false,
		"1\t2":      false,
		"1\n2":      false,
		"1\r2":      false,
	}

	for version, valid := range versions {
		_, err := parseVersion(version)
		if err != nil && valid {
			t.Errorf("Version string '%s' should parse", version)
		}

		if err == nil && !valid {
			t.Errorf("Version string '%s' should not parse", version)
		}
	}
}

// Test complete-version parsing
func TestCompleteVersionParsing(t *testing.T) {
	versions := map[string]*CompleteVersion{
		"1:1.0beta": {Version("1.0beta"), 1, ""},
		"1.0":       {Version("1.0"), 0, ""},
		"2.3-2":     {Version("2.3"), 0, "2"},
		"1::":       nil,
		"4.3--1":    nil,
		"4.1-a":     nil,
		"f:2.3":     nil,
		"1.?":       nil,
	}

	for version, expected := range versions {
		ver, err := NewCompleteVersion(version)
		if err != nil && expected != nil {
			t.Errorf("CompleteVersion string '%s' should not parse", version)
		}

		if err == nil && expected != nil {
			if ver.Version != expected.Version || ver.Epoch != expected.Epoch || ver.Pkgrel != expected.Pkgrel {
				t.Errorf("CompleteVersion string '%s' should parse", version)
			}
		}
	}
}

// Test Newer method
func TestNewer(t *testing.T) {
	a := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("1.0"),
		Pkgrel: "1",
	}
	b := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("2.0"),
		Pkgrel: "1",
	}
	c := &PKGBUILD{
		Epoch:  1,
		Pkgver: Version("1.0"),
		Pkgrel: "1",
	}
	d := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("1.0"),
		Pkgrel: "2",
	}

	if a.Newer(b) {
		t.Errorf("a (%s) should not be newer than b (%s)", a.Version(), b.Version())
	}

	if b.Newer(c) {
		t.Errorf("b (%s) should not be newer than c (%s)", b.Version(), c.Version())
	}

	if a.Newer(d) {
		t.Errorf("a (%s) should not be newer than d (%s)", a.Version(), d.Version())
	}

	if a.Newer(a) {
		t.Errorf("a (%s) should not be newer than itself", a.Version())
	}
}

// Test Older method
func TestOlder(t *testing.T) {
	a := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("1.0"),
		Pkgrel: "1",
	}
	b := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("2.0"),
		Pkgrel: "1",
	}
	c := &PKGBUILD{
		Epoch:  1,
		Pkgver: Version("1.0"),
		Pkgrel: "1",
	}
	d := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("1.0"),
		Pkgrel: "2",
	}

	if !a.Older(b) {
		t.Errorf("a (%s) should be older than b (%s)", a.Version(), b.Version())
	}

	if !b.Older(c) {
		t.Errorf("b (%s) should be older than c (%s)", b.Version(), c.Version())
	}

	if !a.Older(d) {
		t.Errorf("a (%s) should be older than d (%s)", a.Version(), d.Version())
	}

	if d.Older(d) {
		t.Errorf("d (%s) should not be older than itself", d.Version())
	}
}

// Test Version method
func TestVersionMethod(t *testing.T) {
	a := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("1.0"),
		Pkgrel: "1",
	}

	version := "1.0-1"

	if a.Version() != version {
		t.Errorf("a (%s) should be %s", a.Version(), version)
	}

	b := &PKGBUILD{
		Epoch:  4,
		Pkgver: Version("1.0"),
		Pkgrel: "1",
	}

	version = "4:1.0-1"

	if b.Version() != version {
		t.Errorf("a (%s) should be %s", b.Version(), version)
	}
}

// Test random SRCINFO files based on pkgbuilds from Arch core
func TestRandomCoreSRCINFOs(t *testing.T) {
	srcinfos := []string{
		"sudo",
		"pacman",
		"openssh",
		"grub",
		"glibc",
		"systemd",
		"linux",
		"pip2pkgbuild",
		"biicode",
		"teamviewer",
		"shaman-git",
		"bash-snippets",
		"pulseaudio-ctl",
	}

	for _, srcinfo := range srcinfos {
		path := "./test_pkgbuilds/SRCINFO_" + srcinfo
		pkg, err := ParseSRCINFO(path)
		if err != nil {
			t.Errorf("PKGBUILD for %s did not parse: %s", srcinfo, err.Error())
		}

		if pkg.Pkgbase != srcinfo {
			t.Errorf("pkgbase for %s should be %s", srcinfo, pkg.Pkgbase)
		}
	}
}

func TestParseDependency(t *testing.T) {
	deps := make([]*Dependency, 0)
	_, err := parseDependency("linux-mainline-headers<4.6rc1", deps)
	if err != nil {
		t.Errorf("could not parse dependency %s: %s", "bla", err.Error())
	}

	_, err = parseDependency("linux-mainline-headers<=4.6rc1", deps)
	if err != nil {
		t.Errorf("could not parse dependency %s: %s", "bla", err.Error())
	}

	_, err = parseDependency("linux-mainline-headers>=4.6rc1", deps)
	if err != nil {
		t.Errorf("could not parse dependency %s: %s", "bla", err.Error())
	}

	_, err = parseDependency("linux-mainline-headers=4.6rc1", deps)
	if err != nil {
		t.Errorf("could not parse dependency %s: %s", "bla", err.Error())
	}
}

func TestRestrict(t *testing.T) {
	equal := func(a, b *Dependency) bool {
		if a.sgt != b.sgt {
			return false
		}

		if a.slt != b.slt {
			return false
		}

		if a.MaxVer == nil || b.MaxVer == nil {
			if a.MaxVer != b.MaxVer {
				return false
			}
		} else if a.MaxVer.String() != b.MaxVer.String() {
			return false
		}

		if a.MinVer == nil || b.MinVer == nil {
			if a.MinVer != b.MinVer {
				return false
			}
		} else if a.MinVer.String() != b.MinVer.String() {
			return false
		}

		return true
	}

	depStrA := []string{
		"a>=1", "a<=2",
		"b>=1",
		"c>50", "c<99",
		"d>40", "d<50",
		"e>1-1",
		"f>1-1",
		"g>60", "g<100",
		"h>2:1",
	}

	depStrB := []string{
		"a>1", "a<2",
		"b<2",
		"c>1", "c<80",
		"d>1", "d<99",
		"e>1",
		"f>1-2",
		"g=70",
		"h>1:2",
	}

	depStrMerged := []string{
		"a>1", "a<2",
		"b>=1", "b<2",
		"c>50", "c<80",
		"d>40", "d<50",
		"e>1-1",
		"f>1-2",
		"g=70",
		"h>2:1",
	}

	depsA, _ := ParseDeps(depStrA)
	depsB, _ := ParseDeps(depStrB)
	depsMerged, _ := ParseDeps(depStrMerged)

	for i := range depsA {
		RestrictedDep1 := depsA[i].Restrict(depsB[i])
		RestrictedDep2 := depsB[i].Restrict(depsA[i])

		if !equal(depsMerged[i], RestrictedDep1) {
			t.Errorf("%+v should be %+v", RestrictedDep1, depsMerged[i])
		}
		if !equal(depsMerged[i], RestrictedDep2) {
			t.Errorf("%+v should be %+v", RestrictedDep2, depsMerged[i])
		}
	}
}
