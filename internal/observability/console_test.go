package observability

import (
	"bytes"
	"context"
	"testing"

	"github.com/TimCares/go-see"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var testEventID = see.NewID("test.console")

type plainEvent struct{}

func (plainEvent) ID() see.ID { return testEventID }

type messageEvent struct {
	level   see.Level
	message string
}

func (messageEvent) ID() see.ID         { return testEventID }
func (e messageEvent) Level() see.Level { return e.level }
func (e messageEvent) Message() string  { return e.message }

func TestConsoleCorePrintsOnlyFriendlyEventsAtOrAboveLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := zap.New(newConsoleCore(&buf, zapcore.InfoLevel))
	emitter, err := see.New(see.Config{Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	see.SetDefault(emitter)
	t.Cleanup(func() { see.SetDefault(nil) })

	ctx := context.Background()
	see.Emit(ctx, plainEvent{})
	see.Emit(ctx, messageEvent{level: see.LevelInfo, message: ""})
	see.Emit(ctx, messageEvent{level: see.LevelDebug, message: "debug"})
	see.L(ctx).Info("not an event")
	see.Emit(ctx, messageEvent{level: see.LevelInfo, message: "done"})
	see.Emit(ctx, messageEvent{level: see.LevelWarn, message: "careful"})
	see.Emit(ctx, messageEvent{level: see.LevelError, message: "failed"})

	want := "done\nWarning: careful\nError: failed\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}
