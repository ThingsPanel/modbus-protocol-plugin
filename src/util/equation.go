package util

import (
	"fmt"

	"github.com/Knetic/govaluate"
)

func Equation(equation string, param interface{}) (interface{}, error) {
	expr, err := govaluate.NewEvaluableExpression(equation)
	if err != nil {
		return param, err
	}
	parameters := make(map[string]interface{})
	parameters["x"] = param
	result, err := expr.Evaluate(parameters)
	if err != nil {
		return param, err
	}
	fmt.Println(result)
	return result, nil
}
