// Copyright ©2025 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gru implements ASCII file/group/record/unit reading and writing.
package gru

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
)

// ASCII separator control characters.
const (
	FileSeparator   = 0x1c // file separator
	GroupSeparator  = 0x1d // group separator
	RecordSeparator = 0x1e // file separator
	UnitSeparator   = 0x1f // file separator
)

// ErrSeparator is returns by writes containing units that include separator
// characters.
var ErrSeparator = errors.New("unit contains separator")

// File is a collection of groups.
type File []Group

// Group is a collection of records.
type Group []Record

// Record is a collection of units.
type Record []string

// Writer implements ASCII hierarchical file writing.
type Writer struct {
	w io.Writer

	filesStarted   bool
	groupsStarted  bool
	recordsStarted bool
	unitsStarted   bool
}

// NewWriter returns a new Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteFile writes f to the underlying io.Writer.
func (w *Writer) WriteFile(f File) (int, error) {
	var n int
	w.groupsStarted = false
	w.recordsStarted = false
	w.unitsStarted = false
	if w.filesStarted {
		_n, err := w.w.Write([]byte{FileSeparator})
		n += _n
		if err != nil {
			return n, err
		}
	}
	w.filesStarted = true
	for _, g := range f {
		_n, err := w.WriteGroup(g)
		n += _n
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// WriteGroup writes g to the underlying io.Writer.
func (w *Writer) WriteGroup(g Group) (int, error) {
	var n int
	w.recordsStarted = false
	w.unitsStarted = false
	if w.groupsStarted {
		_n, err := w.w.Write([]byte{GroupSeparator})
		n += _n
		if err != nil {
			return n, err
		}
	}
	w.groupsStarted = true
	for _, r := range g {
		_n, err := w.WriteRecord(r)
		n += _n
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// WriteRecord writes r to the underlying io.Writer.
func (w *Writer) WriteRecord(r Record) (int, error) {
	var n int
	w.unitsStarted = false
	if w.recordsStarted {
		_n, err := w.w.Write([]byte{RecordSeparator})
		n += _n
		if err != nil {
			return n, err
		}
	}
	w.recordsStarted = true
	for _, u := range r {
		_n, err := w.WriteUnit(u)
		n += _n
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// WriteUnit writes u to the underlying io.Writer.
func (w *Writer) WriteUnit(u string) (int, error) {
	if strings.ContainsAny(u, "\x1c\x1d\x1e\x1f") {
		return 0, ErrSeparator
	}
	var n int
	if w.unitsStarted {
		_n, err := w.w.Write([]byte{UnitSeparator})
		n += _n
		if err != nil {
			return n, err
		}
	}
	w.unitsStarted = true
	return w.w.Write([]byte(u))
}

// ReadFiles returns a full set of files from the data in r.
func ReadFiles(r io.Reader) ([]File, error) {
	sc := bufio.NewScanner(r)
	sc.Split(ScanFiles)
	var files []File
	for sc.Scan() {
		g, err := ReadGroups(bytes.NewReader(sc.Bytes()))
		files = append(files, g)
		if err != nil {
			return files, err
		}
	}
	err := sc.Err()
	return files, err
}

// ReadGroups returns the set of groups from the data in r up to the next
// file separator.
func ReadGroups(r io.Reader) ([]Group, error) {
	sc := bufio.NewScanner(r)
	sc.Split(ScanGroups)
	var groups []Group
	for sc.Scan() {
		r, err := ReadRecords(bytes.NewReader(sc.Bytes()))
		groups = append(groups, r)
		if err != nil {
			return groups, err
		}
	}
	err := sc.Err()
	return groups, err
}

// ReadRecords returns the set of records from the data in r up to the next
// file or group separator.
func ReadRecords(r io.Reader) ([]Record, error) {
	sc := bufio.NewScanner(r)
	sc.Split(ScanRecords)
	var records []Record
	for sc.Scan() {
		r, err := ReadUnits(bytes.NewReader(sc.Bytes()))
		records = append(records, r)
		if err != nil {
			return records, err
		}
	}
	err := sc.Err()
	return records, err
}

// ReadUnits returns the set of units from the data in r up to the next
// file, group or record separator.
func ReadUnits(r io.Reader) ([]string, error) {
	sc := bufio.NewScanner(r)
	sc.Split(ScanUnit)
	var units []string
	for sc.Scan() {
		units = append(units, sc.Text())
	}
	err := sc.Err()
	return units, err
}

// ScanFiles is a [bufio.SplitFunc] that separates on ASCII FS.
func ScanFiles(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, FileSeparator); i >= 0 {
		// We have a full FS-terminated file.
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated file. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// ScanGroups is a [bufio.SplitFunc] that separates on ASCII GS until the next FS.
func ScanGroups(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, GroupSeparator); i >= 0 {
		// We have a full GS-terminated group.
		return i + 1, data[0:i], nil
	}
	if i := bytes.IndexByte(data, FileSeparator); i >= 0 {
		// We have a full FS-terminated group.
		return i + 1, data[0:i], bufio.ErrFinalToken
	}
	// If we're at EOF, we have a final, non-terminated group. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// ScanRecords is a [bufio.SplitFunc] that separates on ASCII RS until the next
// FS or GS.
func ScanRecords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, RecordSeparator); i >= 0 {
		// We have a full RS-terminated record.
		return i + 1, data[0:i], nil
	}
	if i := bytes.IndexByte(data, GroupSeparator); i >= 0 {
		// We have a full GS-terminated record.
		return i + 1, data[0:i], bufio.ErrFinalToken
	}
	if i := bytes.IndexByte(data, FileSeparator); i >= 0 {
		// We have a full FS-terminated record.
		return i + 1, data[0:i], bufio.ErrFinalToken
	}
	// If we're at EOF, we have a final, non-terminated record. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// ScanUnit is a [bufio.SplitFunc] that separates on ASCII US until the next
// FS, GS or RS.
func ScanUnit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, UnitSeparator); i >= 0 {
		// We have a full RS-terminated unit.
		return i + 1, data[0:i], nil
	}
	if i := bytes.IndexByte(data, RecordSeparator); i >= 0 {
		// We have a full RS-terminated unit.
		return i + 1, data[0:i], bufio.ErrFinalToken
	}
	if i := bytes.IndexByte(data, GroupSeparator); i >= 0 {
		// We have a full GS-terminated unit.
		return i + 1, data[0:i], bufio.ErrFinalToken
	}
	if i := bytes.IndexByte(data, FileSeparator); i >= 0 {
		// We have a full FS-terminated unit.
		return i + 1, data[0:i], bufio.ErrFinalToken
	}
	// If we're at EOF, we have a final, non-terminated unit. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
