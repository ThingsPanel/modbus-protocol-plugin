package util

import (
	"fmt"
	"testing"
)

// Function 2
func TestEquation(t *testing.T) {
	value, err := Equation("2*x", uint64(1024))
	if err != nil {
		t.Error(err.Error())
	} else {
		fmt.Println(value)
		if value.(float64) != float64(2048) {
			t.Fail()
		}
	}
}

func TestFloat32bytes(t *testing.T) {
	var value float32 = 2.0
	b := Float32bytes(value)
	fmt.Println(b)
	if b != nil {
		t.Fail()
	}
}
func TestFloat32frombytes(t *testing.T) {
	var b []byte = []byte{0x40, 0x49, 0x0F, 0xDA}
	value := Float32frombytes(b)
	fmt.Println(value)
	if value != 0 {
		t.Fail()
	}
}
