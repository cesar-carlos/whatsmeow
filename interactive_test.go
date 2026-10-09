package whatsmeow

import (
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestInteractiveMessageContent(t *testing.T) {
	native := &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{Buttons: []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{{Name: proto.String("quick_reply")}}}}}}
	list := &waE2E.Message{ListMessage: &waE2E.ListMessage{ListType: waE2E.ListMessage_SINGLE_SELECT.Enum()}}
	explicit := []waBinary.Node{{Tag: "biz", Attrs: waBinary.Attrs{"custom": "yes"}}, {Tag: "meta", Attrs: waBinary.Attrs{"custom": "yes"}}}
	tests := []struct {
		name  string
		msg   *waE2E.Message
		extra *[]waBinary.Node
		biz   int
		child string
	}{
		{"native", native, nil, 1, "interactive"},
		{"list", list, nil, 1, "list"},
		{"wrapped", &waE2E.Message{DocumentWithCaptionMessage: &waE2E.FutureProofMessage{Message: native}}, nil, 1, "interactive"},
		{"override", list, &explicit, 1, ""},
		{"button response", &waE2E.Message{ButtonsResponseMessage: &waE2E.ButtonsResponseMessage{}}, nil, 0, ""},
		{"list response", &waE2E.Message{ListResponseMessage: &waE2E.ListResponseMessage{}}, nil, 0, ""},
		{"native response", &waE2E.Message{InteractiveResponseMessage: &waE2E.InteractiveResponseMessage{}}, nil, 0, ""},
		{"response override", &waE2E.Message{InteractiveResponseMessage: &waE2E.InteractiveResponseMessage{}}, &explicit, 0, ""},
		{"empty wrapper", &waE2E.Message{DocumentWithCaptionMessage: &waE2E.FutureProofMessage{}}, nil, 0, ""},
		{"nil", nil, nil, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := (*Client)(nil).getMessageContent(waBinary.Node{Tag: "enc"}, tt.msg, waBinary.Attrs{}, false, nodeExtraParams{additionalNodes: tt.extra})
			n := 0
			for _, node := range nodes {
				if node.Tag == "biz" {
					n++
					if tt.child != "" {
						children, ok := node.Content.([]waBinary.Node)
						if !ok || len(children) == 0 || children[0].Tag != tt.child {
							t.Fatalf("wrong biz content: %#v", node)
						}
					}
				}
			}
			if n != tt.biz {
				t.Fatalf("got %d biz nodes, want %d", n, tt.biz)
			}
			if tt.extra != nil {
				found := false
				for _, node := range nodes {
					if node.Tag == "meta" {
						found = true
					}
				}
				if !found {
					t.Fatal("unrelated additional node lost")
				}
			}
		})
	}
}

func TestInteractivePayloadAndFlow(t *testing.T) {
	cyclic := &waE2E.Message{}
	cyclic.EphemeralMessage = &waE2E.FutureProofMessage{Message: cyclic}
	if getButtonTypeFromMessage(cyclic) != "" {
		t.Fatal("cyclic wrapper should not resolve")
	}
	list := &waE2E.Message{ListMessage: &waE2E.ListMessage{ListType: waE2E.ListMessage_SINGLE_SELECT.Enum()}}
	node := interactiveBizNode(list)
	children := node.Content.([]waBinary.Node)
	if children[0].Attrs["type"] != "product_list" {
		t.Fatal("list rendering node missing")
	}
	for _, name := range []string{"payment_info", "review_and_pay", "cta_url", "quick_reply"} {
		t.Run(name, func(t *testing.T) {
			msg := &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{Buttons: []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{{Name: proto.String(name)}}}}}}
			before := proto.Clone(msg)
			node := interactiveBizNode(msg)
			if !proto.Equal(before, msg) {
				t.Fatal("payload mutated")
			}
			if name == "payment_info" || name == "review_and_pay" {
				want := name
				if name == "review_and_pay" {
					want = "order_details"
				}
				if node.Attrs["native_flow_name"] != want {
					t.Fatal("payment flow lost")
				}
			} else {
				children := node.Content.([]waBinary.Node)
				qc := children[1]
				if len(qc.Attrs["decision_id"].(string)) != 40 {
					t.Fatal("invalid decision id")
				}
			}
		})
	}
}
