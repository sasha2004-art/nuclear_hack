package transport

import (
	"encoding/json"
	"testing"
)

func TestWebRTCSignalSerialization(t *testing.T) {
	sig := WebRTCSignalPayload{
		SenderID: "A",
		TargetID: "B",
		Type:     "offer",
		Data:     `{"sdp":"test"}`,
	}

	data, err := json.Marshal(sig)
	if err != nil {
		t.Fatal(err)
	}

	var decoded WebRTCSignalPayload
	json.Unmarshal(data, &decoded)

	if decoded.Type != "offer" || decoded.Data != `{"sdp":"test"}` {
		t.Errorf("Ошибка сериализации сигнала WebRTC")
	}
}

func TestWebRTCSignalTypes(t *testing.T) {
	types := []string{"offer", "answer", "candidate"}

	for _, signalType := range types {
		sig := WebRTCSignalPayload{
			SenderID: "peer1",
			TargetID: "peer2",
			Type:     signalType,
			Data:     `{"test":"data"}`,
		}

		data, _ := json.Marshal(sig)
		var decoded WebRTCSignalPayload
		json.Unmarshal(data, &decoded)

		if decoded.Type != signalType {
			t.Errorf("Expected type %s, got %s", signalType, decoded.Type)
		}
	}
}
