package pkgbuild

import "testing"

// Test version parsing
func TestVersionParsing(t *testing.T) {
	versions := map[string]bool{
		"1.0beta":   true,
		"1.0.0.0.2": true,
		"a.3_4":     true,
		"A.2":       true,
		"_1.2":      false,
		".2":        false,
		"a.2Ø":      false,
		"1.?":       false,
		"1.-":       false,
	}

	for version, valid := range versions {
		_, err := parseVersion(version)
		if err != nil && valid {
			t.Errorf("Version string '%s' should pass", version)
		}

		if err == nil && !valid {
			t.Errorf("Version string '%s' should not pass", version)
		}
	}
}

// Test parsing array value as value
func TestValueParsing(t *testing.T) {
	input := "pkgdesc=([0]=\"value1\" [1]=\"value2\")"

	pkgb, err := parse(input)
	if err != nil {
		t.Error("parse should not fail")
	}

	if pkgb.Pkgdesc != "value1" {
		t.Errorf("should equal 'value1', was: %#v", pkgb.Pkgdesc)
	}
}

// Test parsing array value
func TestArrayValueParsing(t *testing.T) {
	input := "depends=([0]=\"python2\" [1]=\"git\" [2]=\"svn\")"
	depends := []string{
		"python2",
		"git",
		"svn",
	}

	pkgb, err := parse(input)
	if err != nil {
		t.Error("parse should not fail")
	}

	for i, d := range depends {
		if d != pkgb.Depends[i] {
			t.Errorf("should equal '%s', was: '%s'", d, pkgb.Depends[i])
		}
	}
}

// Test Newer method
func TestNewer(t *testing.T) {
	a := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("1.0"),
		Pkgrel: 1,
	}
	b := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("2.0"),
		Pkgrel: 1,
	}
	c := &PKGBUILD{
		Epoch:  1,
		Pkgver: Version("1.0"),
		Pkgrel: 1,
	}
	d := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("1.0"),
		Pkgrel: 2,
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
		Pkgrel: 1,
	}
	b := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("2.0"),
		Pkgrel: 1,
	}
	c := &PKGBUILD{
		Epoch:  1,
		Pkgver: Version("1.0"),
		Pkgrel: 1,
	}
	d := &PKGBUILD{
		Epoch:  0,
		Pkgver: Version("1.0"),
		Pkgrel: 2,
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
		Pkgrel: 1,
	}

	version := "0:1.0-1"

	if a.Version() != version {
		t.Errorf("a (%s) should be %s", a.Version(), version)
	}
}

// Test random pkgbuilds from Arch core
func TestRandomCorePKGBUILDs(t *testing.T) {
	pkgbs := []string{
		"sudo",
		"pacman",
		"openssh",
		"grub",
		"glibc",
		"systemd",
	}

	for _, pkgb := range pkgbs {
		path := "./test_pkgbuilds/PKGBUILD_" + pkgb
		_, err := ParsePKGBUILD(path)
		if err != nil {
			t.Errorf("PKGBUILD for %s did not parse: %s", pkgb, err.Error())
		}
	}
}

// Test parsing source_%src%
func TestArchSourceParsing(t *testing.T) {
	input := "source_x86_64=([0]=\"test-x86_64.tar.gz\")"

	pkgb, err := parse(input)
	if err != nil {
		t.Error("parse should not fail")
	}

	if pkgb.Source[0] != "test-x86_64.tar.gz" {
		t.Errorf("should equal 'value1', was: %#v", pkgb.Pkgdesc)
	}
}
