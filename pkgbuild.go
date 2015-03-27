package pkgbuild

import (
	"bytes"
	"fmt"
	"io/ioutil"
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

// Dependency describes a dependency with min and max version, if any.
type Dependency struct {
	Name   string           // dependency name
	MinVer *CompleteVersion // min version
	sgt    bool             // defines if min version is strictly greater than
	MaxVer *CompleteVersion // max version
	slt    bool             // defines if max version is strictly less than
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
	Pkgnames     []string
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
	Depends      []*Dependency
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
	if p.Epoch > 0 {
		return fmt.Sprintf("%d:%s-%d", p.Epoch, p.Pkgver, p.Pkgrel)
	}

	return fmt.Sprintf("%s-%d", p.Pkgver, p.Pkgrel)
}

// MustParsePKGBUILD must parse the PKGBUILD given by path or it will panic
func MustParsePKGBUILD(path string) *PKGBUILD {
	pkgbuild, err := ParsePKGBUILD(path)
	if err != nil {
		panic(err)
	}
	return pkgbuild
}

// ParsePKGBUILD parses a PKGBUILD given by path.
// Note that this operation is unsafe and should only be used on trusted
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

// MustParseSRCINFO must parse the .SRCINFO given by path or it will panic
func MustParseSRCINFO(path string) *PKGBUILD {
	pkgbuild, err := ParseSRCINFO(path)
	if err != nil {
		panic(err)
	}
	return pkgbuild
}

// ParseSRCINFO parses .SRCINFO file given by path.
// This is a safe alternative to ParsePKGBUILD given that a .SRCINFO file is
// available
func ParseSRCINFO(path string) (*PKGBUILD, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read file: %s, %s", path, err.Error())
	}

	return parsePKGBUILD(string(f))
}

// parse a PKGBUILD and check that the required fields has a non-empty value
func parsePKGBUILD(input string) (*PKGBUILD, error) {
	pkgb, err := parse(input)
	if err != nil {
		return nil, err
	}

	if !validPkgver(string(pkgb.Pkgver)) {
		return nil, fmt.Errorf("invalid pkgver: %s", pkgb.Pkgver)
	}

	if len(pkgb.Arch) == 0 {
		return nil, fmt.Errorf("Arch missing")
	}

	if len(pkgb.Pkgnames) == 0 {
		return nil, fmt.Errorf("missing pkgname")
	}

	for _, name := range pkgb.Pkgnames {
		if !validPkgname(name) {
			return nil, fmt.Errorf("invalid pkgname: %s", name)
		}
	}

	return pkgb, nil
}

// parses a SRCINFO formatted PKGBUILD
func parse(input string) (*PKGBUILD, error) {
	var pkgbuild *PKGBUILD
	var next item

	lexer := lex(input)
Loop:
	for {
		token := lexer.nextItem()
		switch token.typ {
		case itemPkgbase:
			next = lexer.nextItem()
			pkgbuild = &PKGBUILD{Epoch: 0, Pkgbase: next.val}
		case itemPkgname:
			next = lexer.nextItem()
			pkgbuild.Pkgnames = append(pkgbuild.Pkgnames, next.val)
		case itemPkgver:
			next = lexer.nextItem()
			version, err := parseVersion(next.val)
			if err != nil {
				return nil, err
			}
			pkgbuild.Pkgver = version
		case itemPkgrel:
			next = lexer.nextItem()
			rel, err := strconv.ParseInt(next.val, 10, 0)
			if err != nil {
				return nil, err
			}
			pkgbuild.Pkgrel = int(rel)
		case itemPkgdir:
			next = lexer.nextItem()
			pkgbuild.Pkgdir = next.val
		case itemEpoch:
			next = lexer.nextItem()
			epoch, err := strconv.ParseInt(next.val, 10, 0)
			if err != nil {
				return nil, err
			}

			if epoch < 0 {
				return nil, fmt.Errorf("invalid epoch: %d", epoch)
			}
			pkgbuild.Epoch = int(epoch)
		case itemPkgdesc:
			next = lexer.nextItem()
			pkgbuild.Pkgdesc = next.val
		case itemArch:
			next = lexer.nextItem()
			if arch, ok := archs[next.val]; ok {
				pkgbuild.Arch = append(pkgbuild.Arch, arch)
			} else {
				return nil, fmt.Errorf("invalid Arch: %s", next.val)
			}
		case itemURL:
			next = lexer.nextItem()
			pkgbuild.URL = next.val
		case itemLicense:
			next = lexer.nextItem()
			pkgbuild.License = append(pkgbuild.License, next.val)
		case itemGroups:
			next = lexer.nextItem()
			pkgbuild.Groups = append(pkgbuild.Groups, next.val)
		case itemDepends:
			next = lexer.nextItem()
			deps, err := parseDependency(next.val, pkgbuild.Depends)
			if err != nil {
				return nil, err
			}
			pkgbuild.Depends = deps
		case itemOptdepends:
			next = lexer.nextItem()
			pkgbuild.Optdepends = append(pkgbuild.Optdepends, next.val)
		case itemMakedepends:
			next = lexer.nextItem()
			pkgbuild.Makedepends = append(pkgbuild.Makedepends, next.val)
		case itemCheckdepends:
			next = lexer.nextItem()
			pkgbuild.Checkdepends = append(pkgbuild.Checkdepends, next.val)
		case itemProvides:
			next = lexer.nextItem()
			pkgbuild.Provides = append(pkgbuild.Provides, next.val)
		case itemConflicts:
			next = lexer.nextItem()
			pkgbuild.Conflicts = append(pkgbuild.Conflicts, next.val)
		case itemReplaces:
			next = lexer.nextItem()
			pkgbuild.Replaces = append(pkgbuild.Replaces, next.val)
		case itemBackup:
			next = lexer.nextItem()
			pkgbuild.Backup = append(pkgbuild.Backup, next.val)
		case itemOptions:
			next = lexer.nextItem()
			pkgbuild.Options = append(pkgbuild.Options, next.val)
		case itemInstall:
			next = lexer.nextItem()
			pkgbuild.Install = next.val
		case itemChangelog:
			next = lexer.nextItem()
			pkgbuild.Changelog = next.val
		case itemSource:
			next = lexer.nextItem()
			pkgbuild.Source = append(pkgbuild.Source, next.val)
		case itemNoextract:
			next = lexer.nextItem()
			pkgbuild.Noextract = append(pkgbuild.Noextract, next.val)
		case itemMd5sums:
			next = lexer.nextItem()
			pkgbuild.Md5sums = append(pkgbuild.Md5sums, next.val)
		case itemSha1sums:
			next = lexer.nextItem()
			pkgbuild.Sha1sums = append(pkgbuild.Sha1sums, next.val)
		case itemSha224sums:
			next = lexer.nextItem()
			pkgbuild.Sha224sums = append(pkgbuild.Sha224sums, next.val)
		case itemSha256sums:
			next = lexer.nextItem()
			pkgbuild.Sha256sums = append(pkgbuild.Sha256sums, next.val)
		case itemSha384sums:
			next = lexer.nextItem()
			pkgbuild.Sha384sums = append(pkgbuild.Sha384sums, next.val)
		case itemSha512sums:
			next = lexer.nextItem()
			pkgbuild.Sha512sums = append(pkgbuild.Sha512sums, next.val)
		case itemValidpgpkeys:
			next = lexer.nextItem()
			pkgbuild.Validpgpkeys = append(pkgbuild.Validpgpkeys, next.val)
		case itemEndSplit:
		case itemError:
			return nil, fmt.Errorf(token.val)
		case itemEOF:
			break Loop
		default:
			return nil, fmt.Errorf(token.val)
		}
	}
	return pkgbuild, nil
}

// parse and validate a version string
func parseVersion(s string) (Version, error) {
	if validPkgver(s) {
		return Version(s), nil
	}

	return "", fmt.Errorf("invalid version string: %s", s)
}

func parseCompleteVersion(s string) (*CompleteVersion, error) {
	var err error
	epoch := 0
	rel := 0

	// handle possible epoch
	versions := strings.Split(s, ":")
	if len(versions) > 2 {
		return nil, fmt.Errorf("invalid version format: %s", s)
	}

	if len(versions) > 1 {
		epoch, err = strconv.Atoi(versions[0])
		if err != nil {
			return nil, err
		}
	}

	// handle possible rel
	versions = strings.Split(versions[len(versions)-1], "-")
	if len(versions) > 2 {
		return nil, fmt.Errorf("invalid version format: %s", s)
	}

	if len(versions) > 1 {
		rel, err = strconv.Atoi(versions[1])
		if err != nil {
			return nil, err
		}
	}

	// finally check that the actual version is valid
	if validPkgver(versions[0]) {
		return &CompleteVersion{
			Version: Version(versions[0]),
			Epoch:   epoch,
			Pkgrel:  rel,
		}, nil
	}

	return nil, fmt.Errorf("invalid version format: %s", s)
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

// parse dependency with possible version restriction
func parseDependency(dep string, deps []*Dependency) ([]*Dependency, error) {
	var name string
	var dependency *Dependency

	if dep[0] == '-' {
		return nil, fmt.Errorf("invalid dependency name")
	}

	i := 0
	for _, c := range dep {
		if !isValidPkgnameChar(uint8(c)) {
			break
		}
		i++
	}

	// check if the dependency has been set before
	name = dep[0:i]
	for _, d := range deps {
		if d.Name == name {
			dependency = d
		}
	}

	if dependency == nil {
		dependency = &Dependency{
			Name: name,
			sgt:  false,
			slt:  false,
		}
		deps = append(deps, dependency)
	}

	if len(dep) == len(name) {
		return deps, nil
	}

	i++
	var eq bytes.Buffer
	for _, c := range dep[i:] {
		if c != '<' || c != '>' || c != '=' {
			i++
			break
		}
		eq.WriteRune(c)
	}

	version, err := parseCompleteVersion(dep[i:])
	if err != nil {
		return nil, err
	}

	switch eq.String() {
	case "==":
		dependency.MinVer = version
		dependency.MaxVer = version
	case "<=":
		dependency.MaxVer = version
	case ">=":
		dependency.MinVer = version
	case "<":
		dependency.MaxVer = version
		dependency.slt = true
	case ">":
		dependency.MinVer = version
		dependency.sgt = true
	}

	return deps, nil
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
