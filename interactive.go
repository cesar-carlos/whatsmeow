// Copyright (c) 2026 Evolution Go fork contributors.
// Licensed under MPL-2.0, as the rest of whatsmeow.
// Adapted from tulir/whatsmeow PRs #1235 (felps-dev), #1221 (gsdev-br),
// and #1146 (TobyG74). Only interactive node handling is changed.
package whatsmeow

import (
	"encoding/hex"
	"go.mau.fi/util/random"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"strconv"
	"time"
)

// interactivePayload bounds traversal of malformed or cyclic wrappers. It does
// not mutate the message, including ContextInfo and MessageContextInfo.
func interactivePayload(msg *waE2E.Message) *waE2E.Message {
	for depth := 0; msg != nil && depth < 32; depth++ {
		switch {
		case msg.DocumentWithCaptionMessage != nil:
			msg = msg.DocumentWithCaptionMessage.Message
		case msg.ViewOnceMessage != nil:
			msg = msg.ViewOnceMessage.Message
		case msg.ViewOnceMessageV2 != nil:
			msg = msg.ViewOnceMessageV2.Message
		case msg.ViewOnceMessageV2Extension != nil:
			msg = msg.ViewOnceMessageV2Extension.Message
		case msg.EphemeralMessage != nil:
			msg = msg.EphemeralMessage.Message
		default:
			return msg
		}
	}
	return nil
}

func isInteractiveResponse(msg *waE2E.Message) bool {
	return msg != nil && (msg.ButtonsResponseMessage != nil || msg.ListResponseMessage != nil || msg.InteractiveResponseMessage != nil || msg.TemplateButtonReplyMessage != nil)
}

// appendInteractiveNodes gives an explicit biz/hsm node precedence for prompts.
// Replies never carry those nodes; unrelated additional nodes are retained.
func appendInteractiveNodes(content []waBinary.Node, msg *waE2E.Message, additional *[]waBinary.Node) []waBinary.Node {
	msg = interactivePayload(msg)
	response := isInteractiveResponse(msg)
	hasBiz := false
	if additional != nil {
		for _, node := range *additional {
			if node.Tag == "biz" || node.Tag == "hsm" {
				if response || hasBiz {
					continue
				}
				hasBiz = true
			}
			content = append(content, node)
		}
	}
	if !hasBiz && !response {
		if biz := interactiveBizNode(msg); biz != nil {
			content = append(content, *biz)
		}
	}
	return content
}

func interactiveBizNode(msg *waE2E.Message) *waBinary.Node {
	if msg == nil {
		return nil
	}
	kind := getButtonTypeFromMessage(msg)
	if kind == "" {
		return nil
	}
	if kind == "buttons" {
		return &waBinary.Node{Tag: "biz", Content: []waBinary.Node{{Tag: "buttons", Attrs: getButtonAttributes(msg)}}}
	}
	if nf := msg.GetInteractiveMessage().GetNativeFlowMessage(); nf != nil && len(nf.GetButtons()) == 1 {
		name := nf.GetButtons()[0].GetName()
		if name == "payment_info" || name == "review_and_pay" {
			if name == "review_and_pay" {
				name = "order_details"
			}
			return &waBinary.Node{Tag: "biz", Attrs: waBinary.Attrs{"native_flow_name": name}}
		}
	}
	var prompt waBinary.Node
	if kind == "list" {
		// The business node must describe the same list type as the encrypted
		// payload. A single-select menu is not a product catalog.
		prompt = waBinary.Node{Tag: "list", Attrs: getButtonAttributes(msg)}
	} else {
		prompt = waBinary.Node{Tag: "interactive", Attrs: waBinary.Attrs{"type": "native_flow", "v": "1"}, Content: []waBinary.Node{{Tag: "native_flow", Attrs: waBinary.Attrs{"v": "9", "name": "mixed"}}}}
	}
	return &waBinary.Node{Tag: "biz", Attrs: waBinary.Attrs{"actual_actors": "2", "host_storage": "2", "privacy_mode_ts": strconv.FormatInt(time.Now().Unix(), 10)}, Content: []waBinary.Node{
		prompt,
		{Tag: "quality_control", Attrs: waBinary.Attrs{"decision_id": hex.EncodeToString(random.Bytes(20)), "source_type": "third_party"}, Content: []waBinary.Node{{Tag: "decision_source", Attrs: waBinary.Attrs{"value": "df"}}}},
	}}
}
