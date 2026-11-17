package whatsmeow

import (
	"context"
	"fmt"
	"strings"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type statusAckLogger struct {
	waLog.Logger
	warnings []string
}

func (l *statusAckLogger) Warnf(format string, args ...any) {
	l.warnings = append(l.warnings, fmt.Sprintf(format, args...))
}

func TestStatusNodeIsAcknowledgedWithoutBecomingAChat(t *testing.T) {
	own := types.NewJID("5591888888888", types.DefaultUserServer)
	own.Device = 38
	log := &statusAckLogger{Logger: waLog.Noop}
	cli := NewClient(&store.Device{ID: &own}, log)
	cli.SynchronousAck = true
	if cli.nodeHandlers["status"] == nil {
		t.Fatal("status node has no handler")
	}

	var got []*events.Message
	cli.AddEventHandler(func(evt any) {
		if msg, ok := evt.(*events.Message); ok {
			got = append(got, msg)
		}
	})

	ctx := context.Background()
	for _, messageType := range []string{"media", "text"} {
		cli.nodeHandlers["status"](ctx, &waBinary.Node{
			Tag: "status",
			Attrs: waBinary.Attrs{
				"from":        types.StatusBroadcastJID,
				"participant": types.NewJID("5511999999999", types.DefaultUserServer),
				"id":          "STATUS" + messageType,
				"type":        messageType,
				"t":           "1700000000",
			},
		})
		// With no socket, an attempted ACK must report ErrNotConnected. This
		// distinguishes acknowledgement from silently discarding the status.
		want := "Failed to send acknowledgement for status STATUS" + messageType + ": " + ErrNotConnected.Error()
		if len(log.warnings) != 1 || !strings.Contains(log.warnings[0], want) {
			t.Fatalf("status %s did not attempt an ACK: %v", messageType, log.warnings)
		}
		log.warnings = nil
	}
	if len(got) != 0 {
		t.Fatalf("user status became %d message events", len(got))
	}

	cli.handleUnencryptedMessage(ctx, &waBinary.Node{
		Tag: "status",
		Attrs: waBinary.Attrs{
			"from": types.NewJID("120363000000000001", types.NewsletterServer),
			"id":   "NEWS1",
			"type": "text",
			"t":    "1700000000",
		},
		Content: []waBinary.Node{{Tag: "plaintext", Content: []byte{}}},
	})
	if len(got) != 1 {
		t.Fatalf("newsletter status events = %d, want 1", len(got))
	}
	if !got[0].Info.IsNewsletterStatus {
		t.Fatal("newsletter status was not marked")
	}
}
