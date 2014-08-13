/**
 * This file is based on ogórek (https://github.com/kisielk/og-rek).
 * The original copyright and license are retained below.
 *
 * Copyright (c) 2013 Kamil Kisiel
 *
 * Permission is hereby granted, free of charge, to any person
 * obtaining a copy of this software and associated documentation
 * files (the "Software"), to deal in the Software without
 * restriction, including without limitation the rights to use,
 * copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following
 * conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
 * OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
 * HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
 * WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
 * OTHER DEALINGS IN THE SOFTWARE.
 */

package pickle

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
)

const (
	opMark           byte = '('
	opStop                = '.'
	opPop                 = '0'
	opPopMark             = '1'
	opBinfloat            = 'G'
	opBinint              = 'J'
	opBinint1             = 'K'
	opBinint2             = 'M'
	opBinstring           = 'T'
	opShortBinstring      = 'U'
	opBinunicode          = 'X'
	opAppend              = 'a'
	opAppends             = 'e'
	opGet                 = 'g'
	opBinget              = 'h'
	opInst                = 'i'
	opLongBinget          = 'j'
	opList                = 'l'
	opEmptyList           = ']'
	opPut                 = 'p'
	opBinput              = 'q'
	opLongBinput          = 'r'
	opTuple               = 't'
	opEmptyTuple          = ')'

	// Protocol 2
	opProto  = '\x80'
	opTuple1 = '\x85'
	opTuple2 = '\x86'
	opTuple3 = '\x87'
)

var errNotImplemented = errors.New("unimplemented opcode")
var ErrInvalidPickleVersion = errors.New("invalid pickle version")

type OpcodeError struct {
	Key byte
	Pos int
}

func (e OpcodeError) Error() string {
	return fmt.Sprintf("Unknown opcode %d (%c) at position %d: %q", e.Key, e.Key, e.Pos, e.Key)
}

// special marker
type mark struct{}

// None is a representation of Python's None.
type None struct{}

// Decoder is a decoder for pickle streams.
type Decoder struct {
	r     *bufio.Reader
	stack []interface{}
	memo  map[string]interface{}
}

// NewDecoder constructs a new Decoder which will decode the pickle stream in r.
func NewDecoder(r io.Reader) Decoder {
	reader := bufio.NewReader(r)
	return Decoder{reader, make([]interface{}, 0), make(map[string]interface{})}
}

// Decode decodes the pickle stream and returns the result or an error.
func (d Decoder) Decode() (interface{}, error) {

	insn := 0
	for {
		insn++
		key, err := d.r.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		switch key {
		case opMark:
			d.mark()
		case opStop:
			goto done
			break

		case opBinfloat:
			err = d.binFloat()
		case opBinint:
			err = d.loadBinInt()
		case opBinint1:
			err = d.loadBinInt1()
		case opBinint2:
			err = d.loadBinInt2()
		case opBinstring:
			err = d.loadBinString()
		case opShortBinstring:
			err = d.loadShortBinString()
		case opBinunicode:
			err = d.loadBinUnicode()
		case opAppends:
			err = d.loadAppends()
		case opList:
			err = d.loadList()
		case opEmptyList:
			d.push([]interface{}{})
		case opBinput:
			err = d.binPut()
		case opTuple:
			err = d.loadTuple()
		case opTuple1:
			err = d.loadTuple1()
		case opTuple2:
			err = d.loadTuple2()
		case opTuple3:
			err = d.loadTuple3()
		case opEmptyTuple:
			d.push([]interface{}{})

		case opProto:
			v, _ := d.r.ReadByte()
			if v != 2 {
				err = ErrInvalidPickleVersion
			}

		default:
			return nil, OpcodeError{key, insn}
		}

		if err != nil {
			if err == errNotImplemented {
				return nil, OpcodeError{key, insn}
			}
			return nil, err
		}
	}

done:
	return d.pop(), nil
}

// Push a marker
func (d *Decoder) mark() {
	d.push(mark{})
}

// Return the position of the topmost marker
func (d *Decoder) marker() int {
	m := mark{}
	var k int
	for k = len(d.stack) - 1; d.stack[k] != m && k > 0; k-- {
	}
	if k >= 0 {
		return k
	}
	panic("no marker in stack")
}

// Append a new value
func (d *Decoder) push(v interface{}) {
	d.stack = append(d.stack, v)
}

// Pop a value
func (d *Decoder) pop() interface{} {
	ln := len(d.stack) - 1
	v := d.stack[ln]
	d.stack = d.stack[:ln]
	return v
}

// Push a four-byte signed int
func (d *Decoder) loadBinInt() error {
	var b [4]byte
	_, err := io.ReadFull(d.r, b[:])
	if err != nil {
		return err
	}
	v := binary.LittleEndian.Uint32(b[:])
	d.push(int64(v))
	return nil
}

// Push a 1-byte unsigned int
func (d *Decoder) loadBinInt1() error {
	b, _ := d.r.ReadByte()
	d.push(int64(b))
	return nil
}

// Push a 2-byte unsigned int
func (d *Decoder) loadBinInt2() error {
	var b [2]byte
	_, err := io.ReadFull(d.r, b[:])
	if err != nil {
		return err
	}
	v := binary.LittleEndian.Uint16(b[:])
	d.push(int64(v))
	return nil
}

func (d *Decoder) loadBinString() error {
	var b [4]byte
	_, err := io.ReadFull(d.r, b[:])
	if err != nil {
		return err
	}
	v := binary.LittleEndian.Uint32(b[:])
	s := make([]byte, v)
	_, err = io.ReadFull(d.r, s)
	if err != nil {
		return err
	}
	d.push(string(s))
	return nil
}

func (d *Decoder) loadShortBinString() error {
	b, _ := d.r.ReadByte()
	s := make([]byte, b)
	_, err := io.ReadFull(d.r, s)
	if err != nil {
		return err
	}
	d.push(string(s))
	return nil
}

func (d *Decoder) loadBinUnicode() error {
	var length int32
	for i := 0; i < 4; i++ {
		t, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		length = length | (int32(t) << uint(8*i))
	}
	rawB := []byte{}
	for z := 0; int32(z) < length; z++ {
		n, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		rawB = append(rawB, n)
	}
	d.push(string(rawB))
	return nil
}

func (d *Decoder) loadAppends() error {
	k := d.marker()
	l := d.stack[k-1]
	switch l.(type) {
	case []interface{}:
		l := l.([]interface{})
		for _, v := range d.stack[k+1 : len(d.stack)] {
			l = append(l, v)
		}
		d.stack = append(d.stack[:k-1], l)
	default:
		return fmt.Errorf("loadAppends expected a list, got %t", l)
	}
	return nil
}

func (d *Decoder) get() error {
	line, _, err := d.r.ReadLine()
	if err != nil {
		return err
	}
	d.push(d.memo[string(line)])
	return nil
}

func (d *Decoder) binGet() error {
	b, _ := d.r.ReadByte()
	d.push(d.memo[strconv.Itoa(int(b))])
	return nil
}

func (d *Decoder) longBinGet() error {
	var b [4]byte
	_, err := io.ReadFull(d.r, b[:])
	if err != nil {
		return err
	}
	v := binary.LittleEndian.Uint32(b[:])
	d.push(d.memo[strconv.Itoa(int(v))])
	return nil
}

func (d *Decoder) loadList() error {
	k := d.marker()
	v := append([]interface{}{}, d.stack[k+1:]...)
	d.stack = append(d.stack[:k], v)
	return nil
}

func (d *Decoder) loadTuple() error {
	k := d.marker()
	v := append([]interface{}{}, d.stack[k+1:]...)
	d.stack = append(d.stack[:k], v)
	return nil
}

func (d *Decoder) loadTuple1() error {
	k := len(d.stack) - 1
	v := append([]interface{}{}, d.stack[k:]...)
	d.stack = append(d.stack[:k], v)
	return nil
}

func (d *Decoder) loadTuple2() error {
	k := len(d.stack) - 2
	v := append([]interface{}{}, d.stack[k:]...)
	d.stack = append(d.stack[:k], v)
	return nil
}

func (d *Decoder) loadTuple3() error {
	k := len(d.stack) - 3
	v := append([]interface{}{}, d.stack[k:]...)
	d.stack = append(d.stack[:k], v)
	return nil
}

func (d *Decoder) loadPut() error {
	line, _, err := d.r.ReadLine()
	if err != nil {
		return err
	}
	d.memo[string(line)] = d.stack[len(d.stack)-1]
	return nil
}

func (d *Decoder) binPut() error {
	b, _ := d.r.ReadByte()
	d.memo[strconv.Itoa(int(b))] = d.stack[len(d.stack)-1]
	return nil
}

func (d *Decoder) binFloat() error {
	var b [8]byte
	_, err := io.ReadFull(d.r, b[:])
	if err != nil {
		return err
	}
	u := binary.BigEndian.Uint64(b[:])
	d.stack = append(d.stack, math.Float64frombits(u))
	return nil
}
