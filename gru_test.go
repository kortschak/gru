// Copyright ©2025 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gru

import (
	"bytes"
	"reflect"
	"testing"
)

const (
	fs = string(rune(FileSeparator))
	gs = string(rune(GroupSeparator))
	rs = string(rune(RecordSeparator))
	us = string(rune(UnitSeparator))
)

var writerTests = []struct {
	name    string
	files   []File
	wire    []byte
	wantErr error
}{
	{
		name: "full",
		files: []File{
			{
				{
					{"f1_g1_r1_u1", "f1_g1_r1_u2"},
					{"f1_g1_r2_u1", "f1_g1_r2_u2"},
				},
				{
					{"f1_g2_r1_u1", "f1_g2_r1_u2"},
					{"f1_g2_r2_u1", "f1_g2_r2_u2"},
					{"f1_g2_r3_u1", "f1_g2_r3_u2"},
				},
			},
			{
				{
					{"f2_g1_r1_u1", "f2_g1_r1_u2"},
					{"f2_g1_r2_u1", "f2_g1_r2_u2", "f2_g1_r2_u3"},
				},
				{
					{"f2_g2_r1_u1", "f2_g2_r1_u2"},
					{"f2_g2_r2_u1", "f2_g2_r2_u2"},
				},
			},
		},
		wire: []byte("" +
			// file 0
			"f1_g1_r1_u1" + us + "f1_g1_r1_u2" +
			rs +
			"f1_g1_r2_u1" + us + "f1_g1_r2_u2" +

			gs +

			"f1_g2_r1_u1" + us + "f1_g2_r1_u2" +
			rs +
			"f1_g2_r2_u1" + us + "f1_g2_r2_u2" +
			rs +
			"f1_g2_r3_u1" + us + "f1_g2_r3_u2" +

			fs +

			// file 1
			"f2_g1_r1_u1" + us + "f2_g1_r1_u2" +
			rs +
			"f2_g1_r2_u1" + us + "f2_g1_r2_u2" + us + "f2_g1_r2_u3" +

			gs +

			"f2_g2_r1_u1" + us + "f2_g2_r1_u2" +
			rs +
			"f2_g2_r2_u1" + us + "f2_g2_r2_u2",
		),
	},
	{
		name: "csv",
		files: []File{{{
			{"r1_u1", "r1_u2"},
			{"r2_u1", "r2_u2"},
			{"r3_u1", "r3_u2"},
		}}},
		wire: []byte("" +
			"r1_u1" + us + "r1_u2" +
			rs +
			"r2_u1" + us + "r2_u2" +
			rs +
			"r3_u1" + us + "r3_u2",
		),
	},
	{
		name: "invalid_fs",
		files: []File{
			{
				{
					{"f1_g1_r1_u1", "f1_g1_r1_u2"},
					{"f1_g1_" + fs + "r2_u1", "f1_g1_r2_u2"},
				},
			},
		},
		wantErr: ErrSeparator,
	},
	{
		name: "invalid_gs",
		files: []File{
			{
				{
					{"f1_g1_r1_u1", "f1_g1_r1_u2"},
					{"f1_g1_" + gs + "r2_u1", "f1_g1_r2_u2"},
				},
			},
		},
		wantErr: ErrSeparator,
	},
	{
		name: "invalid_rs",
		files: []File{
			{
				{
					{"f1_g1_r1_u1", "f1_g1_r1_u2"},
					{"f1_g1_" + rs + "r2_u1", "f1_g1_r2_u2"},
				},
			},
		},
		wantErr: ErrSeparator,
	},
	{
		name: "invalid_us",
		files: []File{
			{
				{
					{"f1_g1_r1_u1", "f1_g1_r1_u2"},
					{"f1_g1_" + us + "r2_u1", "f1_g1_r2_u2"},
				},
			},
		},
		wantErr: ErrSeparator,
	},
}

func Test(t *testing.T) {
	for _, test := range writerTests {
		t.Run(test.name, func(t *testing.T) {
			t.Run("read", func(t *testing.T) {
				if test.wantErr != nil {
					t.Skip("error testing")
				}
				got, err := ReadFiles(bytes.NewReader(test.wire))
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, test.files) {
					t.Errorf("unexpected result:\ngot: %v\nwant:%v", got, test.files)
				}
			})
			t.Run("write", func(t *testing.T) {
				var got bytes.Buffer
				w := NewWriter(&got)
				for _, f := range test.files {
					_, err := w.WriteFile(f)
					if err != test.wantErr {
						t.Errorf("unexpected error: %v", err)
					}
					if err != nil {
						return
					}
				}
				if !bytes.Equal(got.Bytes(), test.wire) {
					t.Errorf("unexpected result:\ngot: %q\nwant:%q", got.Bytes(), test.wire)
				}
			})
		})
	}
}
