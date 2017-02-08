package index_test

import (
	"github.com/tywkeene/autobd/index"
	"testing"
)

type expect struct {
	Input     string //The input passed to the tested function
	DataEqNil bool   //Should the data returned by the function be nil?
	ErrEqNil  bool   //Should the err returned by the function be nil?
}

func (e *expect) Print(t *testing.T) {
	t.Logf("Input:[%s] DataEqNil:[%v] ErrEqNil:[%v]", e.Input, e.DataEqNil, e.ErrEqNil)
}

func isEmptyIndex(data map[string]*index.Index, shouldBeEmpty bool) bool {
	isEmpty := (len(data) == 0)
	if shouldBeEmpty == true && isEmpty == true {
		return true
	}
	return false
}

func (e *expect) Die(t *testing.T, msg string) {
	e.Print(t)
	t.Fatal(msg)
}

func (e *expect) testError(t *testing.T, err error) {
	switch e.ErrEqNil {
	case true:
		if err != nil {
			e.Die(t, "Err should be nil, but is not:"+err.Error())
		}
	case false:
		if err == nil {
			e.Die(t, "Err should not be nil, but is not")
		}
	}
}

func (e *expect) testIndex(t *testing.T, data map[string]*index.Index) {
	switch e.DataEqNil {
	case true:
		if isEmptyIndex(data, e.DataEqNil) == false {
			e.Die(t, "Index should be empty, but is not")
		}
	case false:
		e.Print(t)
		if isEmptyIndex(data, e.DataEqNil) == true {
			e.Die(t, "Index should not be empty, but is")
		}
	}
}

func TestGetIndex(t *testing.T) {
	var table = []expect{
		//Valid inputs Data should be != nil, err should be == nil
		expect{Input: "./", DataEqNil: false, ErrEqNil: true},
		expect{Input: "/", DataEqNil: false, ErrEqNil: true},
		//Invalid inputs, data should be == nil, err should be != nil
		expect{Input: "./index.go", DataEqNil: true, ErrEqNil: false},
		expect{Input: "../././.asdf", DataEqNil: true, ErrEqNil: false},
		expect{Input: "directoryasdf", DataEqNil: true, ErrEqNil: false},
		expect{Input: "", DataEqNil: true, ErrEqNil: false},
	}
	t.Logf("Running %d tests", len(table))
	for i, test := range table {
		t.Log("Running", i)
		data, err := index.GetIndex(test.Input)
		test.testIndex(t, data)
		test.testError(t, err)
		t.Log("---------------------------------")
	}
}
