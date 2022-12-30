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
