package events

import (
	"github.com/TimCares/go-see"

	"github.com/TimCares/go-dockerctl/internal/config"
)

// ConfigDumpID identifies [ConfigDump] events.
var ConfigDumpID = see.NewID("config.dump")

// ConfigDump dumps the dockerctl config.
//
// [config.Config] never contains secrets directly,
// so no secrets leak into the log.
type ConfigDump struct {
	Cfg config.Config `json:"cfg"`
}

// ID implements [see.Event].
func (ConfigDump) ID() see.ID { return ConfigDumpID }

// Level implements [see.Leveler].
func (e ConfigDump) Level() see.Level {
	return see.LevelInfo
}
