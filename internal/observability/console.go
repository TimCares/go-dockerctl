package observability

import (
	"fmt"
	"io"

	"github.com/TimCares/go-see"
	"go.uber.org/zap/zapcore"
)

// consoleCore prints the friendly message of events emitted through [see.Emit] as plain text.
//
// Every other record, including those of see.L and zap.L, is dropped,
// so only friendly messages are shown in CLI output.
type consoleCore struct {
	zapcore.LevelEnabler
	out io.Writer
}

func newConsoleCore(out io.Writer, level zapcore.LevelEnabler) zapcore.Core {
	return consoleCore{LevelEnabler: level, out: out}
}

// With drops fields, the plain-text line never shows context.
func (c consoleCore) With([]zapcore.Field) zapcore.Core { return c }

func (c consoleCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

// Write only prints if the Event has a valid friendly message.
func (c consoleCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	for _, field := range fields {
		if event, ok := field.Interface.(see.EventObject); ok && event.IsFriendly() {
			_, err := fmt.Fprintln(c.out, levelPrefix(entry.Level)+entry.Message)
			return err
		}
	}
	return nil
}

func (c consoleCore) Sync() error { return nil }

func levelPrefix(level zapcore.Level) string {
	switch {
	case level >= zapcore.ErrorLevel:
		return "Error: "
	case level == zapcore.WarnLevel:
		return "Warning: "
	default:
		return ""
	}
}
