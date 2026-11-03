// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package rustgodecode

import (
	"encoding/binary"
	"errors"
)

var (
	ErrDstTooSmall   = errors.New("varint destination slice too small")
	ErrUnexpectedEOF = errors.New("unexpected EOF while decoding varint")
	ErrOverflow      = errors.New("varint overflows 64-bit unsigned integer")
)

// EncodeVarints encodes values as standard unsigned LEB128 varints.
func EncodeVarints(vals []uint64) []byte {
	out := make([]byte, 0, len(vals)*4)
	for _, v := range vals {
		out = binary.AppendUvarint(out, v)
	}
	return out
}

// DecodeVarints decodes unsigned LEB128 varints from src into dst, returning the count decoded.
// Keeping the loop explicit avoids allocation and mirrors the Rust implementation 1:1.
func DecodeVarints(src []byte, dst []uint64) (int, error) {
	n := 0
	i := 0
	for i < len(src) {
		if n >= len(dst) {
			return n, ErrDstTooSmall
		}
		// Fast path for single-byte varints (0..127), which dominate symbol table refs.
		b := src[i]
		i++
		if b < 0x80 {
			dst[n] = uint64(b)
			n++
			continue
		}
		x := uint64(b & 0x7f)
		shift := uint(7)
		for {
			if i >= len(src) {
				return n, ErrUnexpectedEOF
			}
			b = src[i]
			i++
			x |= uint64(b&0x7f) << shift
			if b < 0x80 {
				break
			}
			shift += 7
			if shift >= 64 {
				return n, ErrOverflow
			}
		}
		dst[n] = x
		n++
	}
	return n, nil
}
