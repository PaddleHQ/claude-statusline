package ratecache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/ratecache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzLoad(f *testing.F) {
	f.Add([]byte(`{"five_hour":{"used":1,"resets_at":2},"seven_day":{"used":3,"resets_at":4}}`))
	f.Add([]byte(`{"five_hour":`))
	f.Add([]byte(`{"five_hour":{"used":1},"seven_day":{"used":3,"resets_at":4}}`))
	f.Add([]byte(`{"five_hour":{"used":-1,"resets_at":2}}`))
	f.Add([]byte(`{"five_hour":{"used":"1","resets_at":2}}`))
	f.Add([]byte(`{"five_hour":{"used":9223372036854775807,"resets_at":1e30}}`))
	f.Add([]byte(`{"five_hour":{"used":1,"resets_at":2},"sessions":{"a":{"five_hour":{"used":1,"resets_at":2}},"b":null,"c":[1]}}`))
	f.Add([]byte(`{"five_hour":{"used":1,"resets_at":2},"sessions":"x"}`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))

	path := filepath.Join(f.TempDir(), "ratelimits.json")

	f.Fuzz(func(t *testing.T, data []byte) {
		require.NoError(t, os.WriteFile(path, data, 0o600))
		limits := ratecache.NewCache(path).Load()
		for name, r := range map[string]*ratecache.Reading{"five_hour": limits.FiveHour, "seven_day": limits.SevenDay} {
			if r == nil {
				continue
			}
			assert.GreaterOrEqual(t, r.Used, int64(0), name)
			assert.GreaterOrEqual(t, r.ResetsAt, int64(0), name)
		}
	})
}
