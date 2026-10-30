package statusline_test

import "testing"

// TestBuildSharesRateLimits pins how the readings in the payload and the shared cache file combine into the bars, and what is written back.
func TestBuildSharesRateLimits(t *testing.T) {
	t.Parallel()

	tests := []scenario{
		{
			name:      `The first rate-limit reading is written to the shared cache file`,
			session:   `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`,
			want:      `Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%`,
			wantCache: `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400},"sessions":{"abc123":{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}}}`,
		},
		{
			name:      `A higher usage reading replaces the cached one and is written back to the cache`,
			session:   `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":55,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`,
			cache:     `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}`,
			want:      `Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 55% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%`,
			wantCache: `{"five_hour":{"used":55,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400},"sessions":{"abc123":{"five_hour":{"used":55,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}}}`,
		},
		{
			name:      `A lower reading from a session not seen before is replaced by the higher cached one`,
			session:   `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":40,"resets_at":1760009000},"seven_day":{"used_percentage":38,"resets_at":1760302400}},"session_id":"abc123"}`,
			cache:     `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}`,
			want:      `Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%`,
			wantCache: `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400},"sessions":{"abc123":{"five_hour":{"used":40,"resets_at":1760009000},"seven_day":{"used":38,"resets_at":1760302400}}}}`,
		},
		{
			name:      `An idle session repeating the same lower reading never lowers the cache`,
			session:   `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":20,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`,
			cache:     `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400},"sessions":{"abc123":{"five_hour":{"used":20,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}}}`,
			want:      `Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%`,
			wantCache: `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400},"sessions":{"abc123":{"five_hour":{"used":20,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}}}`,
		},
		{
			name:      `A session whose reading changed since it last reported is accepted as a real reset`,
			session:   `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":20,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`,
			cache:     `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400},"sessions":{"abc123":{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}}}`,
			want:      `Opus 4.7 [high] \e[2m·\e[0m 5h: \e[33m████\e[0m\e[2m░░░░░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 20% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%`,
			wantCache: `{"five_hour":{"used":20,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400},"sessions":{"abc123":{"five_hour":{"used":20,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}}}`,
		},
		{
			name:      `A reading from an earlier rate-limit window than the cached one is ignored`,
			session:   `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":5,"resets_at":1759991000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`,
			cache:     `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}`,
			want:      `Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m█████████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%`,
			wantCache: `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}`,
		},
		{
			name:      `A reading from a later rate-limit window replaces the cached one even when it is lower`,
			session:   `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":5,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`,
			cache:     `{"five_hour":{"used":90,"resets_at":1759991000},"seven_day":{"used":40,"resets_at":1760302400}}`,
			want:      `Opus 4.7 [high] \e[2m·\e[0m 5h: \e[31m█\e[0m\e[2m░░░░░░░░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 5% \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%`,
			wantCache: `{"five_hour":{"used":5,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400},"sessions":{"abc123":{"five_hour":{"used":5,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}}}`,
		},
		{
			name:    `A rate-limit window whose reset time has passed is hidden`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1759999900},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`,
			want:    `Opus 4.7 [high] \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%`,
		},
	}
	runAll(t, tests)
}
