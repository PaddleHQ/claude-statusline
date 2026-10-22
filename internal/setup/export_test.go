package setup

var (
	Touch        = touch
	InsertMember = insertMember
	RemoveMember = removeMember
	ShellWords   = shellWords
)

func Apply(loc Locations, event, version string) string {
	return apply(config{Locations: loc, event: event, version: version})
}
