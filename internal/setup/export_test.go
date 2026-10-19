package setup

var (
	Touch        = touch
	InsertMember = insertMember
)

func Apply(dir, bin, event, version string) string {
	return apply(config{dir: dir, bin: bin, event: event, version: version})
}
