// Copyright (c) 2022-present, DiceDB contributors
// All rights reserved. Licensed under the BSD 3-Clause License. See LICENSE file in the project root for full license information.

package clientio_test

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"testing"

	"github.com/dicedb/dice/config"
	"github.com/dicedb/dice/internal/clientio"
	"github.com/dicedb/dice/internal/server/utils"
	"github.com/stretchr/testify/assert"
)

func init() {
	parser := config.NewConfigParser()
	if err := parser.ParseDefaults(config.DiceConfig); err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}
}

func TestSimpleStringDecode(t *testing.T) {
	cases := map[string]string{
		"+OK
":                  "OK",
		"+HelloWorld
":        "HelloWorld",
		"+HelloWorldAgain
": "HelloWorldAgain",
	}
	for k, v := range cases {
		p := clientio.NewRESPParser(bytes.NewBuffer([]byte(k)))
		value, err := p.DecodeOne()
		if err != nil {
			t.Error(err)
			t.Errorf("error while decoding: %v", k)
		}
		if v != value {
			t.Fail()
		}
		fmt.Println(v, value)
	}
}

func TestError(t *testing.T) {
	cases := map[string]string{
		"-Error message
": "Error message",
	}
	for k, v := range cases {
		p := clientio.NewRESPParser(bytes.NewBuffer([]byte(k)))
		value, err := p.DecodeOne()
		if err != nil {
			t.Error(err)
			t.Errorf("error while decoding: %v", k)
		}
		if v != value {
			t.Fail()
		}
	}
}

func TestInt64(t *testing.T) {
	cases := map[string]int64{
		":0
":    0,
		":1000
": 1000,
	}
	for k, v := range cases {
		p := clientio.NewRESPParser(bytes.NewBuffer([]byte(k)))
		value, err := p.DecodeOne()
		if err != nil {
			t.Error(err)
			t.Errorf("error while decoding: %v", k)
		}
		if v != value {
			t.Fail()
		}
	}
}

func TestBulkStringDecode(t *testing.T) {
	cases := map[string]string{
		"$5
hello
": "hello",
		"$0

":      utils.EmptyStr,
	}
	for k, v := range cases {
		p := clientio.NewRESPParser(bytes.NewBuffer([]byte(k)))
		value, err := p.DecodeOne()
		if err != nil {
			t.Error(err)
			t.Errorf("error while decoding: %v", k)
		}
		if v != value {
			t.Fail()
		}
	}
}

func TestArrayDecode(t *testing.T) {
	cases := map[string][]interface{}{
		"*0
":                                                   {},
		"*2
$5
hello
$5
world
":                     {"hello", "world"},
		"*3
:1
:2
:3
":                                 {int64(1), int64(2), int64(3)},
		"*5
:1
:2
:3
:4
$5
hello
":            {int64(1), int64(2), int64(3), int64(4), "hello"},
		"*2
*3
:1
:2
:3
*2
+Hello
-World
": {[]int64{int64(1), int64(2), int64(3)}, []interface{}{"Hello", "World"}},
	}
	for k, v := range cases {
		p := clientio.NewRESPParser(bytes.NewBuffer([]byte(k)))
		value, err := p.DecodeOne()
		if err != nil {
			t.Error(err)
			t.Errorf("error while decoding: %v", v)
		}
		array := value.([]interface{})
		if len(array) != len(v) {
			t.Fail()
		}
		for i := range array {
			if fmt.Sprintf("%v", v[i]) != fmt.Sprintf("%v", array[i]) {
				t.Fail()
			}
		}
	}
}

func TestSimpleStrings(t *testing.T) {
	var b []byte
	var buf = bytes.NewBuffer(b)
	for i := 0; i < 1024; i++ {
		buf.WriteByte('a' + byte(i%26))
		e := clientio.Encode(buf.String(), true)
		p := clientio.NewRESPParser(bytes.NewBuffer(e))
		nv, err := p.DecodeOne()
		if err != nil {
			t.Error(err)
			t.Errorf("resp parser test failed for value: %v", buf.Bytes())
		}
		if nv != buf.String() {
			t.Errorf("resp parser decoded value mismatch: %v", buf.String())
		}
	}
}

func TestBulkStrings(t *testing.T) {
	var b []byte
	var buf = bytes.NewBuffer(b)
	for i := 0; i < 1024; i++ {
		buf.WriteByte('a' + byte(i%26))
		e := clientio.Encode(buf.String(), false)
		p := clientio.NewRESPParser(bytes.NewBuffer(e))
		nv, err := p.DecodeOne()
		if err != nil {
			t.Error(err)
			t.Errorf("resp parser test failed for value: %v", buf.Bytes())
		}
		if nv != buf.String() {
			t.Errorf("resp parser decoded value mismatch: %v", buf.String())
		}
	}
}

func TestInt(t *testing.T) {
	for _, v := range []int64{math.MinInt8, math.MinInt16, math.MinInt32, math.MinInt64, 0, math.MaxInt8, math.MaxInt16, math.MaxInt32, math.MaxInt64} {
		e := clientio.Encode(v, false)
		p := clientio.NewRESPParser(bytes.NewBuffer(e))
		nv, err := p.DecodeOne()
		if err != nil {
			t.Error(err)
			t.Errorf("resp parser test failed for value: %v", v)
		}
		if nv != v {
			t.Errorf("resp parser decoded value mismatch: %v", v)
		}
	}
}

func TestArrayInt(t *testing.T) {
	var b []byte
	var buf = bytes.NewBuffer(b)
	for i := 0; i < 1024; i++ {
		buf.WriteByte('a' + byte(i%26))
		e := clientio.Encode(buf.String(), true)
		p := clientio.NewRESPParser(bytes.NewBuffer(e))
		nv, err := p.DecodeOne()
		if err != nil {
			t.Error(err)
			t.Errorf("resp parser test failed for value: %v", buf.Bytes())
		}
		if nv != buf.String() {
			t.Errorf("resp parser decoded value mismatch: %v", buf.String())
		}
	}
}

func TestBoolean(t *testing.T) {
	tests := []struct {
		input  bool
		output []byte
	}{
		{
			input:  true,
			output: []byte("+true
"),
		},
		{
			input:  false,
			output: []byte("+false
"),
		},
	}

	for _, v := range tests {
		ev := clientio.Encode(v.input, false)
		assert.Equal(t, ev, v.output)
	}
}

func TestInteger(t *testing.T) {
	tests := []struct {
		input  int
		output []byte
	}{
		{
			input:  10,
			output: []byte(":10
"),
		},
		{
			input:  -19,
			output: []byte(":-19
"),
		},
	}

	for _, v := range tests {
		ev := clientio.Encode(v.input, false)
		assert.Equal(t, ev, v.output)
	}
}
