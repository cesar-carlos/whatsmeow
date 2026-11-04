package whatsmeow

import (
	"context"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func TestStatusNodeIsAcknowledgedWithoutBecomingAChat(t *testing.T) {
	own := types.NewJID("5591888888888", types.DefaultUserServer)
	own.Device = 38
	cli := NewClient(&store.Device{ID: &own}, waLog.Noop)
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
		cli.handleUnencryptedMessage(ctx, &waBinary.Node{
			Tag: "status",
			Attrs: waBinary.Attrs{
				"from":        types.StatusBroadcastJID,
				"participant": types.NewJID("5511999999999", types.DefaultUserServer),
				"id":          "STATUS" + messageType,
				"type":        messageType,
				"t":           "1700000000",
			},
		})
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
