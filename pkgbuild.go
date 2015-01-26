package pkgbuild

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Arch is a system architecture
type Arch int

const (
	// Any architecture
	Any Arch = iota
	// I686 architecture
	I686
	// X8664 x86_64 (64bit) architecture
	X8664
	// ARMv5 architecture (archlinux-arm)
	ARMv5
	// ARMv6h architecture (archlinux-arm)
	ARMv6h
	// ARMv7h architecture (archlinux-arm)
	ARMv7h
)

var archs = map[string]Arch{
	"any":    Any,
	"i686":   I686,
	"x86_64": X8664,
	"armv7h": ARMv7h,
}

// PKGBUILD is a struct describing a parsed PKGBUILD file.
// Required fields are:
//	pkgname
//	pkgver
//	pkgrel
//	arch
//	(license) - not required but recommended
//
// parsing a PKGBUILD file without these fields will fail
type PKGBUILD struct {
	Pkgname      string  // required
	Pkgver       Version // required
	Pkgrel       int     // required
	Pkgdir       string
	Epoch        int
	Pkgbase      string
	Pkgdesc      string
	Arch         []Arch // required
	URL          string
	License      []string // recommended
	Groups       []string
	Depends      []string
	Optdepends   []string
	Makedepends  []string
	Checkdepends []string
	Provides     []string
	Conflicts    []string
	Replaces     []string
	Backup       []string
	Options      []string
	Install      string
	Changelog    string
	Source       []string
	Noextract    []string
	Md5sums      []string
	Sha1sums     []string
	Sha224sums   []string
	Sha256sums   []string
	Sha384sums   []string
	Sha512sums   []string
	Validpgpkeys []string
}

// Newer is true if p has a higher version number than p2
func (p *PKGBUILD) Newer(p2 *PKGBUILD) bool {
	if p.Epoch < p2.Epoch {
		return false
	}

	if p.Pkgver.bigger(p2.Pkgver) {
		return true
	}

	if p2.Pkgver.bigger(p.Pkgver) {
		return false
	}

	return p.Pkgrel > p2.Pkgrel
}

// Older is true if p has a smaller version number than p2
func (p *PKGBUILD) Older(p2 *PKGBUILD) bool {
	if p.Epoch < p2.Epoch {
		return true
	}

	if p2.Pkgver.bigger(p.Pkgver) {
		return true
	}

	if p.Pkgver.bigger(p2.Pkgver) {
		return false
	}

	return p.Pkgrel < p2.Pkgrel
}

// Version returns the full version of the PKGBUILD (including epoch and rel)
func (p *PKGBUILD) Version() string {
	return fmt.Sprintf("%d:%s-%d", p.Epoch, p.Pkgver, p.Pkgrel)
}

// MustParsePKGBUILD must parse the PKGBUILD or it will panic
func MustParsePKGBUILD(path string) *PKGBUILD {
	pkgbuild, err := ParsePKGBUILD(path)
	if err != nil {
		panic(err)
	}
	return pkgbuild
}

// ParsePKGBUILD parses a PKGBUILD given by path
// note that this operation is unsafe and should only be used on trusted
// PKGBUILDs or within some kind of jail, e.g. a VM, container or chroot
func ParsePKGBUILD(path string) (*PKGBUILD, error) {
	// TODO parse maintainer if possible (read first x bytes of the file)
	// check for valid path
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s: no such file", path)
		}
		return nil, err
	}

	source := fmt.Sprintf("source %s && set", path)
	out, err := exec.Command("bash", "-c", source).Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("unable to parse PKGBUILD: %s", path)
		}
		return nil, err
	}

	return parsePKGBUILD(string(out))
}

// parse a PKGBUILD and check that the required fields has a non-empty value
func parsePKGBUILD(input string) (*PKGBUILD, error) {
	pkgbuild, err := parse(input)
	if err != nil {
		return nil, err
	}

	if !validPkgname(pkgbuild.Pkgname) {
		return nil, fmt.Errorf("invalid pkgname '%s'", pkgbuild.Pkgname)
	}

	if !validPkgver(string(pkgbuild.Pkgver)) {
		return nil, fmt.Errorf("invalid pkgver '%s'", pkgbuild.Pkgver)
	}

	if len(pkgbuild.Arch) == 0 {
		return nil, fmt.Errorf("Arch missing")
	}

	return pkgbuild, nil
}

// parses a sourced PKGBUILD
func parse(input string) (*PKGBUILD, error) {
	pkgbuild := &PKGBUILD{Epoch: 0}
	lexer := lex(input)
Loop:
	for {
		token := lexer.nextItem()
		switch token.typ {
		case itemPkgname:
			pkgbuild.Pkgname = parseValue(lexer)
		case itemPkgver:
			next := lexer.nextItem()
			version, err := parseVersion(next.val)
			if err != nil {
				return nil, err
			}
			pkgbuild.Pkgver = version
		case itemPkgrel:
			next := lexer.nextItem()
			rel, err := strconv.ParseInt(next.val, 10, 0)
			if err != nil {
				return nil, err
			}
			pkgbuild.Pkgrel = int(rel)
		case itemPkgdir:
			pkgbuild.Pkgdir = parseValue(lexer)
		case itemEpoch:
			next := lexer.nextItem()
			epoch, err := strconv.ParseInt(next.val, 10, 0)
			if err != nil {
				return nil, err
			}

			if epoch < 0 {
				return nil, fmt.Errorf("invalid epoch %d", epoch)
			}
			pkgbuild.Epoch = int(epoch)
		case itemPkgbase:
			pkgbuild.Pkgbase = parseValue(lexer)
		case itemPkgdesc:
			pkgbuild.Pkgdesc = parseValue(lexer)
		case itemArch:
			val, err := parseArchs(lexer)
			if err != nil {
				return nil, err
			}
			pkgbuild.Arch = val
		case itemURL:
			pkgbuild.URL = parseValue(lexer)
		case itemLicense:
			pkgbuild.License = parseArrayValues(lexer)
		case itemGroups:
			pkgbuild.Groups = parseArrayValues(lexer)
		case itemDepends:
			pkgbuild.Depends = parseArrayValues(lexer)
		case itemOptdepends:
			pkgbuild.Optdepends = parseArrayValues(lexer)
		case itemMakedepends:
			pkgbuild.Makedepends = parseArrayValues(lexer)
		case itemCheckdepends:
			pkgbuild.Checkdepends = parseArrayValues(lexer)
		case itemProvides:
			pkgbuild.Provides = parseArrayValues(lexer)
		case itemConflicts:
			pkgbuild.Conflicts = parseArrayValues(lexer)
		case itemReplaces:
			pkgbuild.Replaces = parseArrayValues(lexer)
		case itemBackup:
			pkgbuild.Backup = parseArrayValues(lexer)
		case itemOptions:
			pkgbuild.Options = parseArrayValues(lexer)
		case itemInstall:
			pkgbuild.Install = parseValue(lexer)
		case itemChangelog:
			pkgbuild.Changelog = parseValue(lexer)
		case itemSource:
			pkgbuild.Source = parseArrayValues(lexer)
		case itemNoextract:
			pkgbuild.Noextract = parseArrayValues(lexer)
		case itemMd5sums:
			pkgbuild.Md5sums = parseArrayValues(lexer)
		case itemSha1sums:
			pkgbuild.Sha1sums = parseArrayValues(lexer)
		case itemSha224sums:
			pkgbuild.Sha224sums = parseArrayValues(lexer)
		case itemSha256sums:
			pkgbuild.Sha256sums = parseArrayValues(lexer)
		case itemSha384sums:
			pkgbuild.Sha384sums = parseArrayValues(lexer)
		case itemSha512sums:
			pkgbuild.Sha512sums = parseArrayValues(lexer)
		case itemValidpgpkeys:
			pkgbuild.Validpgpkeys = parseArrayValues(lexer)
		case itemEOF:
			break Loop
		case itemError:
			return nil, fmt.Errorf(token.val)
		}
	}
	return pkgbuild, nil
}

// parse a value to a correctly formatted string
func parseValue(l *lexer) string {
	switch token := l.nextItem(); token.typ {
	case itemValue:
		return strings.Replace(token.val, "'\\''", "'", -1)
	case itemArrayValue:
		// discard all the next array items of the current array
		for l.nextItem().typ != itemArrayEnd {
		}
		return parseArrayValue(token.val)
	default:
		return ""
	}
}

// parse array values into a string array
func parseArrayValues(l *lexer) []string {
	array := []string{}
Loop:
	for {
		switch next := l.nextItem(); next.typ {
		case itemArrayValue:
			array = append(array, parseArrayValue(next.val))
		case itemArrayEnd:
			break Loop
		}
	}
	return array
}

// parse a bash array value
func parseArrayValue(v string) string {
	return strings.Replace(v, "\\\"", "\"", -1)
}

// parse and validate a version string
func parseVersion(s string) (Version, error) {
	if validPkgver(s) {
		return Version(s), nil
	}

	return "", fmt.Errorf("invalid version string '%s'", s)
}

// parse archs into an Arch array
func parseArchs(l *lexer) ([]Arch, error) {
	array := []Arch{}
Loop:
	for {
		switch next := l.nextItem(); next.typ {
		case itemArrayValue:
			if arch, ok := archs[next.val]; ok {
				array = append(array, arch)
			} else {
				return nil, errors.New("invalid Arch: " + next.val)
			}
		case itemArrayEnd:
			break Loop
		}
	}
	return array, nil
}

// check if name is a valid pkgname format
func validPkgname(name string) bool {
	if len(name) < 1 {
		return false
	}

	if name[0] == '-' {
		return false
	}

	for _, r := range name {
		if !isValidPkgnameChar(uint8(r)) {
			return false
		}
	}

	return true
}

// check if version is a valid pkgver format
func validPkgver(version string) bool {
	if len(version) < 1 {
		return false
	}

	if !isAlphaNumeric(version[0]) {
		return false
	}

	for _, r := range version[1:] {
		if !isValidPkgverChar(uint8(r)) {
			return false
		}
	}

	return true
}

// isLowerAlpha reports whether c is a lowercase alpha character
func isLowerAlpha(c uint8) bool {
	return 'a' <= c && c <= 'z'
}

// check if c is a valid pkgname char
func isValidPkgnameChar(c uint8) bool {
	return isLowerAlpha(c) || isDigit(c) || c == '@' || c == '.' || c == '_' || c == '+' || c == '-'
}

// check if c is a valid pkgver char
func isValidPkgverChar(c uint8) bool {
	return isAlphaNumeric(c) || c == '_' || c == '+' || c == '.'
}
