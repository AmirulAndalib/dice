// Copyright (c) 2022-present, DiceDB contributors
// All rights reserved. Licensed under the BSD 3-Clause License. See LICENSE file in the project root for full license information.

package netconn

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNetConnIOHandler_RESP(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedRead     string
		writeResponse    string
		expectedWrite    string
		readErr          error
		writeErr         error
		ctxTimeout       time.Duration
		expectedReadErr  error
		expectedWriteErr error
	}{
		{
			name:          "Simple String",
			input:         "+OK
",
			expectedRead:  "+OK
",
			writeResponse: "+OK
",
			expectedWrite: "+OK
",
		},
		{
			name:          "Error",
			input:         "-Error message
",
			expectedRead:  "-Error message
",
			writeResponse: "-ERR unknown command 'FOOBAR'
",
			expectedWrite: "-ERR unknown command 'FOOBAR'
",
		},
		{
			name:          "Integer",
			input:         ":1000
",
			expectedRead:  ":1000
",
			writeResponse: ":1000
",
			expectedWrite: ":1000
",
		},
		{
			name:          "Bulk String",
			input:         "$5
hello
",
			expectedRead:  "$5
hello
",
			writeResponse: "$5
world
",
			expectedWrite: "$5
world
",
		},
		{
			name:          "Null Bulk String",
			input:         "$-1
",
			expectedRead:  "$-1
",
			writeResponse: "$-1
",
			expectedWrite: "$-1
",
		},
		{
			name:          "Empty Bulk String",
			input:         "$0

",
			expectedRead:  "$0

",
			writeResponse: "$0

",
			expectedWrite: "$0

",
		},
		{
			name:          "Array",
			input:         "*2
$5
hello
$5
world
",
			expectedRead:  "*2
$5
hello
$5
world
",
			writeResponse: "*2
$5
hello
$5
world
",
			expectedWrite: "*2
$5
hello
$5
world
",
		},
		{
			name:          "Empty Array",
			input:         "*0
",
			expectedRead:  "*0
",
			writeResponse: "*0
",
			expectedWrite: "*0
",
		},
		{
			name:          "Null Array",
			input:         "*-1
",
			expectedRead:  "*-1
",
			writeResponse: "*-1
",
			expectedWrite: "*-1
",
		},
		{
			name:          "Nested Array",
			input:         "*2
*2
+foo
+bar
*2
+hello
+world
",
			expectedRead:  "*2
*2
+foo
+bar
*2
+hello
+world
",
			writeResponse: "*2
*2
+foo
+bar
*2
+hello
+world
",
			expectedWrite: "*2
*2
+foo
+bar
*2
+hello
+world
",
		},
		{
			name:          "SET command",
			input:         "*3
$3
SET
$3
key
$5
value
",
			expectedRead:  "*3
$3
SET
$3
key
$5
value
",
			writeResponse: "+OK
",
			expectedWrite: "+OK
",
		},
		{
			name:          "GET command",
			input:         "*2
$3
GET
$3
key
",
			expectedRead:  "*2
$3
GET
$3
key
",
			writeResponse: "$5
value
",
			expectedWrite: "$5
value
",
		},
		{
			name:          "LPUSH command",
			input:         "*4
$5
LPUSH
$4
list
$5
value
$6
value2
",
			expectedRead:  "*4
$5
LPUSH
$4
list
$5
value
$6
value2
",
			writeResponse: ":2
",
			expectedWrite: ":2
",
		},
		{
			name:          "HMSET command",
			input:         "*6
$5
HMSET
$4
hash
$5
field
$5
value
$6
field2
$6
value2
",
			expectedRead:  "*6
$5
HMSET
$4
hash
$5
field
$5
value
$6
field2
$6
value2
",
			writeResponse: "+OK
",
			expectedWrite: "+OK
",
		},
		{
			name:          "Partial read",
			input:         "*2
$5
hello
$5
wor",
			expectedRead:  "*2
$5
hello
$5
wor",
			writeResponse: "+OK
",
			expectedWrite: "+OK
",
		},
		{
			name:            "Read error",
			input:           "*2
$5
hello
$5
world
",
			readErr:         errors.New("read error"),
			expectedReadErr: errors.New("error reading request: read error"),
		},
		{
			name:             "Write error",
			input:            "*2
$5
hello
$5
world
",
			expectedRead:     "*2
$5
hello
$5
world
",
			writeResponse:    strings.Repeat("Hello, World!
", 100),
			writeErr:         errors.New("write error"),
			expectedWriteErr: errors.New("error writing response: write error"),
		},
		{
			name:             "Write error",
			input:            "*2
$5
hello
$5
world
",
			expectedRead:     "*2
$5
hello
$5
world
",
			writeResponse:    "Hello, World!
",
			writeErr:         errors.New("write error"),
			expectedWriteErr: errors.New("error writing response: write error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockConn{
				readData: []byte(tt.input),
				readErr:  tt.readErr,
				writeErr: tt.writeErr,
			}

			handler := &IOHandler{
				conn:   mock,
				reader: bufio.NewReaderSize(mock, 512),
				writer: bufio.NewWriterSize(mock, 1024),
				readPool: &sync.Pool{
					New: func() interface{} {
						b := make([]byte, ioBufferSize)
						return &b // Return pointer
					},
				},
			}

			ctx := context.Background()
			if tt.ctxTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.ctxTimeout)
				defer cancel()
			}

			// Test ReadRequest
			data, err := handler.Read(ctx)
			if tt.expectedReadErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedReadErr.Error(), err.Error())
				return
			} else {
				assert.NoError(t, err)
				assert.Equal(t, []byte(tt.expectedRead), data)
			}

			// Test WriteResponse
			err = handler.Write(ctx, []byte(tt.writeResponse))
			if tt.expectedWriteErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedWriteErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, []byte(tt.expectedWrite), mock.writeData.Bytes())
			}
		})
	}
}
