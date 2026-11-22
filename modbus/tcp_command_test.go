package modbus

import (
	"reflect"
	"testing"
)

func TestParseTCPResponsePreservesUnitIdentifier(t *testing.T) {
	command := TCPCommand{RequestTransactionID: 0x1234}
	response := []byte{
		0x12, 0x34, // transaction ID
		0x00, 0x00, // protocol ID
		0x00, 0x05, // length: unit ID + PDU
		0x01, 0x03, 0x02, 0x00, 0x2a, // unit ID, FC03, byte count, register value 42
	}

	got, err := command.ParseTCPResponse(response)
	if err != nil {
		t.Fatalf("ParseTCPResponse() error = %v", err)
	}
	want := []byte{0x01, 0x03, 0x02, 0x00, 0x2a}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseTCPResponse() = % x, want % x", got, want)
	}
}
