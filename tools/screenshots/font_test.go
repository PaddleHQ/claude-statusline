package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const repoRoot = "../.."

// TestSymbolsFontCoversEveryIcon catches an icon added to the code but not to the font subset, which the
// documentation images would otherwise draw as an empty box.
func TestSymbolsFontCoversEveryIcon(t *testing.T) {
	t.Parallel()

	covered, err := cmapRunes(symbolsFont)
	require.NoError(t, err)

	used := iconLiterals(t, repoRoot)
	require.NotEmpty(t, used, "found no Private Use Area code points in the Go code")
	for r, where := range used {
		assert.True(t, covered[r], "U+%04X, used in %s, is missing from tools/screenshots/symbols.woff: see tools/screenshots/FONT.md", r, where)
	}
}

// iconLiterals maps each Private Use Area rune in the string and rune literals of the non-test Go files under
// root to one file that uses it.
func iconLiterals(t *testing.T, root string) map[rune]string {
	t.Helper()

	used := map[rune]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || (lit.Kind != token.STRING && lit.Kind != token.CHAR) {
				return true
			}
			value, err := strconv.Unquote(lit.Value)
			require.NoError(t, err, "%s: %s", path, lit.Value)
			for _, r := range value {
				if isIcon(r) {
					used[r] = path
				}
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)
	return used
}

// cmapRunes returns the code points a WOFF font maps, read from its format 12 cmap subtable.
func cmapRunes(woff []byte) (map[rune]bool, error) {
	cmap, err := woffTable(woff, "cmap")
	if err != nil {
		return nil, err
	}
	if len(cmap) < 4 {
		return nil, fmt.Errorf("cmap is %d bytes", len(cmap))
	}
	be := binary.BigEndian
	for i := range int(be.Uint16(cmap[2:])) {
		rec := 4 + 8*i
		if rec+8 > len(cmap) {
			return nil, fmt.Errorf("cmap record %d is out of bounds", i)
		}
		sub := int(be.Uint32(cmap[rec+4:]))
		if sub+16 > len(cmap) || be.Uint16(cmap[sub:]) != 12 {
			continue
		}
		groups := int(be.Uint32(cmap[sub+12:]))
		if sub+16+12*groups > len(cmap) {
			return nil, fmt.Errorf("cmap format 12 groups are out of bounds")
		}
		runes := map[rune]bool{}
		for g := range groups {
			at := sub + 16 + 12*g
			for r := be.Uint32(cmap[at:]); r <= be.Uint32(cmap[at+4:]); r++ {
				runes[rune(r)] = true
			}
		}
		return runes, nil
	}
	return nil, fmt.Errorf("no format 12 cmap subtable")
}

// woffTable returns the uncompressed bytes of one table of a WOFF 1.0 font.
func woffTable(woff []byte, tag string) ([]byte, error) {
	const headerSize, entrySize = 44, 20
	be := binary.BigEndian
	if len(woff) < headerSize || string(woff[:4]) != "wOFF" {
		return nil, fmt.Errorf("not a WOFF 1.0 font")
	}
	for i := range int(be.Uint16(woff[12:])) {
		entry := headerSize + entrySize*i
		if entry+entrySize > len(woff) {
			return nil, fmt.Errorf("table directory entry %d is out of bounds", i)
		}
		if string(woff[entry:entry+4]) != tag {
			continue
		}
		offset, compLength, origLength := be.Uint32(woff[entry+4:]), be.Uint32(woff[entry+8:]), be.Uint32(woff[entry+12:])
		if uint64(offset)+uint64(compLength) > uint64(len(woff)) {
			return nil, fmt.Errorf("table %s is out of bounds", tag)
		}
		data := woff[offset : offset+compLength]
		if compLength == origLength {
			return data, nil
		}
		z, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("table %s: %w", tag, err)
		}
		return io.ReadAll(io.LimitReader(z, int64(origLength)))
	}
	return nil, fmt.Errorf("no %s table", tag)
}
