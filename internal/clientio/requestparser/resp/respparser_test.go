// Copyright (c) 2022-present, DiceDB contributors
// All rights reserved. Licensed under the BSD 3-Clause License. See LICENSE file in the project root for full license information.

package respparser

import (
	"reflect"
	"testing"

	"github.com/dicedb/dice/internal/cmd"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []*cmd.DiceDBCmd
		wantErr bool
	}{
		{
			name:  "Simple SET command",
			input: "*3
$3
SET
$3
key
$5
value
",
			want: []*cmd.DiceDBCmd{
				{Cmd: "SET", Args: []string{"key", "value"}},
			},
		},
		{
			name:  "GET command",
			input: "*2
$3
GET
$3
key
",
			want: []*cmd.DiceDBCmd{
				{Cmd: "GET", Args: []string{"key"}},
			},
		},
		{
			name:  "Multiple commands",
			input: "*2
$4
PING
$4
PONG
*3
$3
SET
$3
key
$5
value
",
			want: []*cmd.DiceDBCmd{
				{Cmd: "PING", Args: []string{"PONG"}},
				{Cmd: "SET", Args: []string{"key", "value"}},
			},
		},
		{
			name:  "Command with integer argument",
			input: "*3
$6
EXPIRE
$3
key
:60
",
			want: []*cmd.DiceDBCmd{
				{Cmd: "EXPIRE", Args: []string{"key", "60"}},
			},
		},
		{
			name:    "Invalid command (not an array)",
			input:   "NOT AN ARRAY
",
			wantErr: true,
		},
		{
			name:    "Empty command",
			input:   "*0
",
			wantErr: true,
		},
		{
			name:  "Command with null bulk string argument",
			input: "*3
$3
SET
$3
key
$-1
",
			want: []*cmd.DiceDBCmd{
				{Cmd: "SET", Args: []string{"key", "(nil)"}},
			},
		},
		{
			name:  "Command with Simple String argument",
			input: "*3
$3
SET
$3
key
+OK
",
			want: []*cmd.DiceDBCmd{
				{Cmd: "SET", Args: []string{"key", "OK"}},
			},
		},
		{
			name:  "Command with Error argument",
			input: "*3
$3
SET
$3
key
-ERR Invalid argument
",
			want: []*cmd.DiceDBCmd{
				{Cmd: "SET", Args: []string{"key", "ERR Invalid argument"}},
			},
		},
		{
			name:  "Command with mixed argument types",
			input: "*5
$4
MSET
$3
key
$5
value
:1000
+OK
",
			want: []*cmd.DiceDBCmd{
				{Cmd: "MSET", Args: []string{"key", "value", "1000", "OK"}},
			},
		},
		{
			name:    "Invalid array length",
			input:   "*-2
$3
SET
$3
key
$5
value
",
			wantErr: true,
		},
		{
			name:    "Incomplete command",
			input:   "*3
$3
SET
$3
key
",
			wantErr: true,
		},
		{
			name:  "Command with empty bulk string",
			input: "*3
$3
SET
$3
key
$0

",
			want: []*cmd.DiceDBCmd{
				{Cmd: "SET", Args: []string{"key", ""}},
			},
		},
		{
			name:    "Invalid bulk string length",
			input:   "*3
$3
SET
$3
key
$-2
value
",
			wantErr: true,
		},
		{
			name:    "Non-integer bulk string length",
			input:   "*3
$3
SET
$3
key
$abc
value
",
			wantErr: true,
		},
		{
			name:  "Large bulk string",
			input: "*2
$4
ECHO
$1000
" + string(make([]byte, 1000)) + "
",
			want: []*cmd.DiceDBCmd{
				{Cmd: "ECHO", Args: []string{string(make([]byte, 1000))}},
			},
		},
		{
			name:    "Incomplete CRLF",
			input:   "*2
$4
ECHO
$5
hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got, err := p.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_parseSimpleString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"Valid simple string", "+OK
", "OK", false},
		{"Empty simple string", "+
", "", false},
		{"Simple string with spaces", "+Hello World
", "Hello World", false},
		{"Incomplete simple string", "+OK", "", true},
		{"Missing CR", "+OK
", "", true},
		{"Missing LF", "+OK", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{data: []byte(tt.input), pos: 0}
			got, err := p.parseSimpleString()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSimpleString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseSimpleString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_parseError(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"Valid error", "-Error message
", "Error message", false},
		{"Empty error", "-
", "", false},
		{"Error with spaces", "-ERR unknown command
", "ERR unknown command", false},
		{"Incomplete error", "-Error", "", true},
		{"Missing CR", "-Error
", "", true},
		{"Missing LF", "-Error", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{data: []byte(tt.input), pos: 0}
			got, err := p.parseError()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_parseInteger(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"Valid positive integer", ":1000
", 1000, false},
		{"Valid negative integer", ":-1000
", -1000, false},
		{"Zero", ":0
", 0, false},
		{"Large integer", ":9223372036854775807
", 9223372036854775807, false},
		{"Invalid integer (float)", ":3.14
", 0, true},
		{"Invalid integer (text)", ":abc
", 0, true},
		{"Incomplete integer", ":123", 0, true},
		{"Missing CR", ":123
", 0, true},
		{"Missing LF", ":123", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{data: []byte(tt.input), pos: 0}
			got, err := p.parseInteger()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInteger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseInteger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_parseBulkString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"Valid bulk string", "$5
hello
", "hello", false},
		{"Empty bulk string", "$0

", "", false},
		{"Null bulk string", "$-1
", "(nil)", false},
		{"Bulk string with spaces", "$11
hello world
", "hello world", false},
		{"Invalid length (negative)", "$-2
hello
", "", true},
		{"Invalid length (non-numeric)", "$abc
hello
", "", true},
		{"Incomplete bulk string", "$5
hell", "", true},
		{"Missing CR", "$5
hello
", "", true},
		{"Missing LF", "$5
hello", "", true},
		{"Length mismatch", "$4
hello
", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{data: []byte(tt.input), pos: 0}
			got, err := p.parseBulkString()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBulkString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseBulkString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_parseArray(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "Valid array",
			input: "*2
$5
hello
$5
world
",
			want:  []string{"hello", "world"},
		},
		{
			name:    "Empty array",
			input:   "*0
",
			wantErr: true,
		},
		{
			name:    "Null array",
			input:   "*-1
",
			wantErr: true,
		},
		{
			name:  "Array with mixed types",
			input: "*3
:1
$5
hello
+world
",
			want:  []string{"1", "hello", "world"},
		},
		{
			name:    "Invalid array length",
			input:   "*-2
",
			wantErr: true,
		},
		{
			name:    "Non-numeric array length",
			input:   "*abc
",
			wantErr: true,
		},
		{
			name:    "Array length mismatch (too few elements)",
			input:   "*3
$5
hello
$5
world
",
			wantErr: true,
		},
		{
			name:  "Array length mismatch (too many elements)",
			input: "*1
$5
hello
$5
world
",
			want:  []string{"hello"}, // Truncated parsing
		},
		{
			name:    "Incomplete array",
			input:   "*2
$5
hello
",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{data: []byte(tt.input), pos: 0}
			got, err := p.parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
