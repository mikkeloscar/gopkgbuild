package pkgbuild

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
func MustParsePKGBUILD(path string) []*PKGBUILD {
	pkgbuild, err := ParsePKGBUILD(path)
	if err != nil {
		panic(err)
	}
	return pkgbuild
}

// ParsePKGBUILD parses a PKGBUILD given by path
// note that this operation is unsafe and should only be used on trusted
// PKGBUILDs or within some kind of jail, e.g. a VM, container or chroot
func ParsePKGBUILD(path string) ([]*PKGBUILD, error) {
	// TODO parse maintainer if possible (read first x bytes of the file)
	// check for valid path
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s: no such file", path)
		}
		return nil, err
	}

	// depend on pkgbuild-introspection (mksrcinfo)
	out, err := exec.Command("/usr/bin/mksrcinfo", "-o", "/dev/stdout", path).Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("unable to parse PKGBUILD: %s", path)
		}
		return nil, err
	}

	return parsePKGBUILD(string(out))
}

// parse a PKGBUILD and check that the required fields has a non-empty value
func parsePKGBUILD(input string) ([]*PKGBUILD, error) {
	pkgbuilds, err := parse(input)
	if err != nil {
		return nil, err
	}

	for _, pkgb := range pkgbuilds {
		if !validPkgname(pkgb.Pkgname) {
			return nil, fmt.Errorf("invalid pkgname: %s", pkgb.Pkgname)
		}

		if !validPkgver(string(pkgb.Pkgver)) {
			return nil, fmt.Errorf("invalid pkgver: %s", pkgb.Pkgver)
		}

		if len(pkgb.Arch) == 0 {
			return nil, fmt.Errorf("Arch missing")
		}
	}

	return pkgbuilds, nil
}

func parsePackage(l *lexer, pkgbuild *PKGBUILD) (*PKGBUILD, error) {
	var next item
	for {
		token := l.nextItem()
		switch token.typ {
		case itemPkgver:
			next = l.nextItem()
			version, err := parseVersion(next.val)
			if err != nil {
				return nil, err
			}
			pkgbuild.Pkgver = version
		case itemPkgrel:
			next = l.nextItem()
			rel, err := strconv.ParseInt(next.val, 10, 0)
			if err != nil {
				return nil, err
			}
			pkgbuild.Pkgrel = int(rel)
		case itemPkgdir:
			next = l.nextItem()
			pkgbuild.Pkgdir = next.val
		case itemEpoch:
			next = l.nextItem()
			epoch, err := strconv.ParseInt(next.val, 10, 0)
			if err != nil {
				return nil, err
			}

			if epoch < 0 {
				return nil, fmt.Errorf("invalid epoch: %d", epoch)
			}
			pkgbuild.Epoch = int(epoch)
		case itemPkgdesc:
			next = l.nextItem()
			pkgbuild.Pkgdesc = next.val
		case itemArch:
			next = l.nextItem()
			if arch, ok := archs[next.val]; ok {
				pkgbuild.Arch = append(pkgbuild.Arch, arch)
			} else {
				return nil, fmt.Errorf("invalid Arch: %s", next.val)
			}
		case itemURL:
			next = l.nextItem()
			pkgbuild.URL = next.val
		case itemLicense:
			next = l.nextItem()
			pkgbuild.License = append(pkgbuild.License, next.val)
		case itemGroups:
			next = l.nextItem()
			pkgbuild.Groups = append(pkgbuild.Groups, next.val)
		case itemDepends:
			next = l.nextItem()
			pkgbuild.Depends = append(pkgbuild.Depends, next.val)
		case itemOptdepends:
			next = l.nextItem()
			pkgbuild.Optdepends = append(pkgbuild.Optdepends, next.val)
		case itemMakedepends:
			next = l.nextItem()
			pkgbuild.Makedepends = append(pkgbuild.Makedepends, next.val)
		case itemCheckdepends:
			next = l.nextItem()
			pkgbuild.Checkdepends = append(pkgbuild.Checkdepends, next.val)
		case itemProvides:
			next = l.nextItem()
			pkgbuild.Provides = append(pkgbuild.Provides, next.val)
		case itemConflicts:
			next = l.nextItem()
			pkgbuild.Conflicts = append(pkgbuild.Conflicts, next.val)
		case itemReplaces:
			next = l.nextItem()
			pkgbuild.Replaces = append(pkgbuild.Replaces, next.val)
		case itemBackup:
			next = l.nextItem()
			pkgbuild.Backup = append(pkgbuild.Backup, next.val)
		case itemOptions:
			next = l.nextItem()
			pkgbuild.Options = append(pkgbuild.Options, next.val)
		case itemInstall:
			next = l.nextItem()
			pkgbuild.Install = next.val
		case itemChangelog:
			next = l.nextItem()
			pkgbuild.Changelog = next.val
		case itemSource:
			next = l.nextItem()
			pkgbuild.Source = append(pkgbuild.Source, next.val)
		case itemNoextract:
			next = l.nextItem()
			pkgbuild.Noextract = append(pkgbuild.Noextract, next.val)
		case itemMd5sums:
			next = l.nextItem()
			pkgbuild.Md5sums = append(pkgbuild.Md5sums, next.val)
		case itemSha1sums:
			next = l.nextItem()
			pkgbuild.Sha1sums = append(pkgbuild.Sha1sums, next.val)
		case itemSha224sums:
			next = l.nextItem()
			pkgbuild.Sha224sums = append(pkgbuild.Sha224sums, next.val)
		case itemSha256sums:
			next = l.nextItem()
			pkgbuild.Sha256sums = append(pkgbuild.Sha256sums, next.val)
		case itemSha384sums:
			next = l.nextItem()
			pkgbuild.Sha384sums = append(pkgbuild.Sha384sums, next.val)
		case itemSha512sums:
			next = l.nextItem()
			pkgbuild.Sha512sums = append(pkgbuild.Sha512sums, next.val)
		case itemValidpgpkeys:
			next = l.nextItem()
			pkgbuild.Validpgpkeys = append(pkgbuild.Validpgpkeys, next.val)
		case itemEndSplit:
			return pkgbuild, nil
		case itemError:
			return nil, fmt.Errorf(token.val)
		}
	}
}

// parses a sourced PKGBUILD
func parse(input string) ([]*PKGBUILD, error) {
	var pkgbase *PKGBUILD
	var pkgbuild *PKGBUILD
	var err error
	var next item

	pkgbuilds := []*PKGBUILD{}
	lexer := lex(input)
Loop:
	for {
		token := lexer.nextItem()
		switch token.typ {
		case itemPkgbase:
			next = lexer.nextItem()
			pkgbase = &PKGBUILD{Epoch: 0, Pkgbase: next.val}
			pkgbase, err = parsePackage(lexer, pkgbase)
			if err != nil {
				return nil, err
			}
		case itemPkgname:
			next = lexer.nextItem()
			pkgb := *pkgbase
			pkgb.Pkgname = next.val
			pkgbuild, err = parsePackage(lexer, &pkgb)
			if err != nil {
				return nil, err
			}
			pkgbuilds = append(pkgbuilds, pkgbuild)
		case itemEOF:
			break Loop
		default:
			fmt.Println(token.val)
		}
	}
	return pkgbuilds, nil
}

// parse and validate a version string
func parseVersion(s string) (Version, error) {
	if validPkgver(s) {
		return Version(s), nil
	}

	return "", fmt.Errorf("invalid version string: %s", s)
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
