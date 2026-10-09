package whatsmeow

import (
	"reflect"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestWrappedListPreservesSelectionContract(t *testing.T) {
	wrappers := map[string]func(*waE2E.Message) *waE2E.Message{
		"direct": func(m *waE2E.Message) *waE2E.Message { return m },
		"document": func(m *waE2E.Message) *waE2E.Message {
			return &waE2E.Message{DocumentWithCaptionMessage: &waE2E.FutureProofMessage{Message: m}}
		},
		"view-once": func(m *waE2E.Message) *waE2E.Message {
			return &waE2E.Message{ViewOnceMessage: &waE2E.FutureProofMessage{Message: m}}
		},
		"view-once-v2": func(m *waE2E.Message) *waE2E.Message {
			return &waE2E.Message{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: m}}
		},
		"view-once-extension": func(m *waE2E.Message) *waE2E.Message {
			return &waE2E.Message{ViewOnceMessageV2Extension: &waE2E.FutureProofMessage{Message: m}}
		},
		"ephemeral": func(m *waE2E.Message) *waE2E.Message {
			return &waE2E.Message{EphemeralMessage: &waE2E.FutureProofMessage{Message: m}}
		},
	}
	for _, listType := range []waE2E.ListMessage_ListType{waE2E.ListMessage_SINGLE_SELECT, waE2E.ListMessage_PRODUCT_LIST} {
		for name, wrap := range wrappers {
			t.Run(listType.String()+"/"+name, func(t *testing.T) {
				payload := &waE2E.Message{ListMessage: &waE2E.ListMessage{
					ListType:    listType.Enum(),
					Sections:    []*waE2E.ListMessage_Section{{Rows: []*waE2E.ListMessage_Row{{RowID: proto.String("chosen-row")}}}},
					ContextInfo: &waE2E.ContextInfo{StanzaID: proto.String("quoted"), MentionedJID: []string{"123@s.whatsapp.net"}},
				}, MessageContextInfo: &waE2E.MessageContextInfo{MessageSecret: []byte{1, 2, 3}}}
				msg := wrap(payload)
				before := proto.Clone(msg)
				nodes := (*Client)(nil).getMessageContent(waBinary.Node{Tag: "enc"}, msg, waBinary.Attrs{}, false, nodeExtraParams{})
				if len(nodes) != 2 || nodes[1].Tag != "biz" {
					t.Fatalf("unexpected content: %#v", nodes)
				}
				child := nodes[1].Content.([]waBinary.Node)[0]
				want := "single_select"
				if listType == waE2E.ListMessage_PRODUCT_LIST {
					want = "product_list"
				}
				if child.Tag != "list" || child.Attrs["type"] != want || child.Attrs["v"] != "2" {
					t.Fatalf("list payload %s advertised as %#v", listType, child)
				}
				if attrs := getButtonAttributes(msg); attrs["type"] != want {
					t.Fatalf("wrapped attributes: %#v", attrs)
				}
				if !proto.Equal(before, msg) {
					t.Fatal("selection IDs, context or wrapper changed")
				}
			})
		}
	}
}

func TestMalformedInteractiveAttributesAreBounded(t *testing.T) {
	cycle := &waE2E.Message{}
	cycle.ViewOnceMessage = &waE2E.FutureProofMessage{Message: cycle}
	deep := &waE2E.Message{ButtonsMessage: &waE2E.ButtonsMessage{}}
	for i := 0; i < 40; i++ {
		deep = &waE2E.Message{EphemeralMessage: &waE2E.FutureProofMessage{Message: deep}}
	}
	for _, msg := range []*waE2E.Message{nil, {}, {ViewOnceMessage: &waE2E.FutureProofMessage{}}, cycle, deep} {
		if len(getButtonAttributes(msg)) != 0 {
			t.Fatal("malformed wrapper returned attributes")
		}
		nodes := (*Client)(nil).getMessageContent(waBinary.Node{Tag: "enc"}, msg, waBinary.Attrs{}, false, nodeExtraParams{})
		if len(nodes) != 1 {
			t.Fatal("malformed payload received a business node")
		}
	}
}

func TestResponseNodesAndExplicitOverrides(t *testing.T) {
	additional := []waBinary.Node{{Tag: "biz"}, {Tag: "hsm"}, {Tag: "biz"}, {Tag: "meta", Attrs: waBinary.Attrs{"keep": "yes"}}}
	before := append([]waBinary.Node(nil), additional...)
	for _, response := range []*waE2E.Message{
		{ButtonsResponseMessage: &waE2E.ButtonsResponseMessage{}},
		{ListResponseMessage: &waE2E.ListResponseMessage{}},
		{InteractiveResponseMessage: &waE2E.InteractiveResponseMessage{}},
		{TemplateButtonReplyMessage: &waE2E.TemplateButtonReplyMessage{}},
	} {
		msg := &waE2E.Message{ViewOnceMessageV2Extension: &waE2E.FutureProofMessage{Message: response}}
		nodes := (*Client)(nil).getMessageContent(waBinary.Node{Tag: "enc"}, msg, waBinary.Attrs{}, false, nodeExtraParams{additionalNodes: &additional})
		if len(nodes) != 2 || nodes[1].Tag != "meta" {
			t.Fatalf("response retained business nodes or lost metadata: %#v", nodes)
		}
	}
	msg := &waE2E.Message{ListMessage: &waE2E.ListMessage{ListType: waE2E.ListMessage_SINGLE_SELECT.Enum()}}
	nodes := (*Client)(nil).getMessageContent(waBinary.Node{Tag: "enc"}, msg, waBinary.Attrs{}, false, nodeExtraParams{additionalNodes: &additional})
	if len(nodes) != 3 || nodes[1].Tag != "biz" || nodes[2].Tag != "meta" {
		t.Fatalf("duplicate override: %#v", nodes)
	}
	if !reflect.DeepEqual(before, additional) {
		t.Fatal("caller-owned nodes mutated")
	}
}
