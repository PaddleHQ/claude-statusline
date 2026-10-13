// Package transcript reads the few facts the status line takes from a session transcript.
//
// Transcripts can be enormous, so only the first and last few lines are read, each within a byte bound.
package transcript

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

const (
	headLines = 40
	tailLines = 200
	// headBytes, subheadBytes and tailBytes are backstops: reading normally stops after a few kilobytes.
	headBytes    = 4 << 20
	subheadBytes = 1 << 20
	tailBytes    = 4 << 20
	chunkBytes   = 64 << 10
	headChunk    = 8 << 10
	// maxLine is the longest line that is kept; a longer one is read past without being held in memory.
	maxLine      = 256 << 10
	maxSubagents = 64
)

var (
	modelPattern = regexp.MustCompile(`"message":\{"model":"([^"]*)"`)
	bareSkill    = regexp.MustCompile(`^- [^ ]+$`)
)

// Info is what a transcript reveals; every field is zero when it is absent or unreadable.
type Info struct {
	// ActualModel is the model of the latest assistant turn in the last 200 lines.
	ActualModel string
	// Boot is the prompt size of the first assistant turn.
	Boot int64
	// SkillsChars is the length of the first skill listing.
	SkillsChars int
	// SkillsBare and SkillsTotal count the skills in the initial listing that lost their description, and all listed.
	SkillsBare, SkillsTotal int
	// MemDropped reports a SessionStart hook whose output was too large to inject.
	MemDropped bool
	// SubBare and SubTotal are the same counts for the subagent with the most stripped skills.
	SubBare, SubTotal int
}

// source is the part of a file the reader needs.
type source interface {
	io.Reader
	io.ReaderAt
	io.Closer
	Stat() (fs.FileInfo, error)
}

type reader struct {
	open func(string) (source, error)
}

func openFile(path string) (source, error) {
	f, err := os.Open(path) //nolint:gosec // G304: reading the transcript path Claude Code passed is the feature
	if err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	return f, nil
}

// Read gathers Info from the transcript at path. An empty path or any read error yields the zero Info.
func Read(path string) Info { return reader{open: openFile}.read(path) }

func (r reader) read(path string) Info {
	if path == "" {
		return Info{}
	}
	f, err := r.open(path)
	if err != nil {
		return Info{}
	}
	defer func() { _ = f.Close() }()
	stat, err := f.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		return Info{}
	}

	var out Info
	opening, boot := head(f, headBytes, true)
	out.Boot = boot
	out.SkillsChars = skillsChars(opening)
	out.MemDropped = memDropped(opening)
	out.SkillsBare, out.SkillsTotal, _ = skillCounts(opening)
	out.ActualModel = lastModel(f, stat.Size())
	out.SubBare, out.SubTotal = r.subagents(path)
	return out
}

// head streams lines from the start of src and stops once it has the opening lines, and, when wantBoot is set,
// the first assistant line. It returns the opening lines that hold attachments and the prompt size of that assistant line.
func head(src io.Reader, limit int64, wantBoot bool) (lines []string, boot int64) {
	br := bufio.NewReaderSize(io.LimitReader(src, limit), headChunk)
	for n := 0; moreHead(n, wantBoot, boot); n++ {
		line, ok := nextLine(br)
		if !ok {
			break
		}
		if n < headLines && strings.Contains(line, `"attachment"`) {
			lines = append(lines, line)
		}
		if wantBoot && boot == 0 && strings.Contains(line, `"type":"assistant"`) {
			boot = bootOf(line)
			wantBoot = false
		}
	}
	return lines, boot
}

func moreHead(n int, wantBoot bool, boot int64) bool {
	return n < headLines || (wantBoot && boot == 0 && n < maxBootSearch)
}

// maxBootSearch bounds how many lines are read looking for the first assistant turn.
const maxBootSearch = 1000

// nextLine returns the next line, or "" for one longer than maxLine. ok is false at the end of the input or a cut-off line.
func nextLine(br *bufio.Reader) (line string, ok bool) {
	var buf []byte
	tooLong := false
	for {
		chunk, err := br.ReadSlice('\n')
		if !tooLong {
			if len(buf)+len(chunk) > maxLine {
				tooLong, buf = true, nil
			} else {
				buf = append(buf, chunk...)
			}
		}
		switch err {
		case nil:
			return strings.TrimSuffix(string(buf), "\n"), true
		case bufio.ErrBufferFull:
		default:
			// A final line with no newline may be cut by the byte bound, so it is not trusted.
			return "", false
		}
	}
}

// lastModel finds the model of the last assistant turn within the last 200 lines, reading backwards from the end.
func lastModel(f io.ReaderAt, size int64) string {
	var pending []byte
	off, seen := size, 0
	trailing := true
	for off > 0 && size-off < tailBytes {
		n := min(chunkBytes, off)
		off -= n
		buf := make([]byte, n+int64(len(pending)))
		if _, err := f.ReadAt(buf[:n], off); err != nil && err != io.EOF {
			return ""
		}
		copy(buf[n:], pending)
		pending = nil

		complete := buf
		if off > 0 {
			before, after, ok := bytes.Cut(buf, []byte{'\n'})
			if !ok {
				pending = buf
				continue
			}
			pending, complete = before, after
		}
		if trailing {
			complete, trailing = bytes.TrimSuffix(complete, []byte("\n")), false
		}
		if m, done := scanBackwards(complete, &seen); done {
			return m
		}
		if seen >= tailLines {
			return ""
		}
	}
	return ""
}

// scanBackwards looks through complete lines from the last to the first. It reports done once a model is found
// or the 200-line window is used up.
func scanBackwards(data []byte, seen *int) (string, bool) {
	for end := len(data); end >= 0; {
		start := bytes.LastIndexByte(data[:end], '\n') + 1
		if *seen >= tailLines {
			return "", true
		}
		*seen++
		if m := modelPattern.FindSubmatch(data[start:end]); m != nil {
			return string(m[1]), true
		}
		end = start - 1
	}
	return "", false
}

func bootOf(line string) int64 {
	if !gjson.Valid(line) {
		return 0
	}
	u := gjson.Get(line, "message.usage")
	if !u.Exists() {
		return 0
	}
	sum := u.Get("input_tokens").Int() + u.Get("cache_creation_input_tokens").Int() + u.Get("cache_read_input_tokens").Int()
	return max(sum, 0)
}

func attachments(lines []string, kind string) []gjson.Result {
	var out []gjson.Result
	for _, l := range lines {
		if !gjson.Valid(l) {
			continue
		}
		if a := gjson.Get(l, "attachment"); a.Get("type").String() == kind {
			out = append(out, a)
		}
	}
	return out
}

func skillsChars(lines []string) int {
	listings := attachments(lines, "skill_listing")
	if len(listings) == 0 {
		return 0
	}
	return utf8.RuneCountInString(listings[0].Get("content").String())
}

// skillCounts reads the first initial skill listing: how many entries lost their description, and how many there are.
func skillCounts(lines []string) (bare, total int, ok bool) {
	for _, a := range attachments(lines, "skill_listing") {
		if a.Get("isInitial").Type != gjson.True {
			continue
		}
		for entry := range strings.SplitSeq(a.Get("content").String(), "\n") {
			if !strings.HasPrefix(entry, "- ") {
				continue
			}
			total++
			if bareSkill.MatchString(entry) {
				bare++
			}
		}
		return bare, total, true
	}
	return 0, 0, false
}

func memDropped(lines []string) bool {
	for _, a := range attachments(lines, "hook_success") {
		if a.Get("hookEvent").String() == "SessionStart" && strings.Contains(a.Get("content").String(), "<persisted-output>") {
			return true
		}
	}
	return false
}

// subagents finds the subagent transcript with the most stripped skills.
func (r reader) subagents(path string) (bare, total int) {
	files, _ := filepath.Glob(filepath.Join(strings.TrimSuffix(path, ".jsonl"), "subagents", "*.jsonl"))
	for _, name := range files[:min(len(files), maxSubagents)] {
		f, err := r.open(name)
		if err != nil {
			continue
		}
		lines, _ := head(f, subheadBytes, false)
		_ = f.Close()
		if b, t, ok := skillCounts(lines); ok && (total == 0 || b > bare) {
			bare, total = b, t
		}
	}
	return bare, total
}
