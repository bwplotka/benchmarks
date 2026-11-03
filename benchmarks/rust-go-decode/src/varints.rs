// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

//! Unsigned LEB128 varint decoder matching `varints.go` 1:1.

use std::fmt;

/// Errors returned when decoding an unsigned LEB128 varint stream.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum VarintError {
    DstTooSmall,
    UnexpectedEof,
    Overflow,
}

impl fmt::Display for VarintError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::DstTooSmall => f.write_str("varint destination slice too small"),
            Self::UnexpectedEof => f.write_str("unexpected EOF while decoding varint"),
            Self::Overflow => f.write_str("varint overflows 64-bit unsigned integer"),
        }
    }
}

impl std::error::Error for VarintError {}

/// Decodes unsigned LEB128 varints from `src` into `dst`, returning the number of values written.
pub fn decode_varints(src: &[u8], dst: &mut [u64]) -> Result<usize, VarintError> {
    let mut n = 0usize;
    let mut i = 0usize;
    while i < src.len() {
        if n >= dst.len() {
            return Err(VarintError::DstTooSmall);
        }
        // Fast path for single-byte varints (0..127), which dominate symbol table refs.
        let mut b = src[i];
        i += 1;
        if b < 0x80 {
            dst[n] = u64::from(b);
            n += 1;
            continue;
        }
        let mut x = u64::from(b & 0x7f);
        let mut shift = 7u32;
        loop {
            if i >= src.len() {
                return Err(VarintError::UnexpectedEof);
            }
            b = src[i];
            i += 1;
            x |= u64::from(b & 0x7f) << shift;
            if b < 0x80 {
                break;
            }
            shift += 7;
            if shift >= 64 {
                return Err(VarintError::Overflow);
            }
        }
        dst[n] = x;
        n += 1;
    }
    Ok(n)
}
