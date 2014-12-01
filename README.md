# goPKGBUILD

A golang package for parsing Arch Linux PKGBUILDs

## Usage

Example usage

```go
package main

import (
    "fmt"

    "github.com/mikkeloscar/gopkgbuild"
)

func main() {
    pkgb, err := ParsePKGBUILD("/path/to/PKGBUILD")
    if err != nil {
        fmt.Println(err)
    }

    fmt.Printf("Package name: %s", pkgb.Pkgname)
}
```

## LICENSE

Copyright (C) 2014  Mikkel Oscar Lyderik Larsen

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
