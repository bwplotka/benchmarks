// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

pub mod proto;
pub mod varints;

use prost::Message;

pub use varints::{VarintError, decode_varints};

/// Structural counts and sample checksums matching `DecodeSummary` in `remotewrite.go`.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DecodeSummary {
    pub uncompressed_bytes: usize,
    pub compressed_bytes: usize,
    pub timeseries: usize,
    pub samples: usize,
    pub histograms: usize,
    pub exemplars: usize,
    pub symbols: usize,
    pub value_bits_xor: u64,
    pub sample_value_sum: f64,
}

/// Decompresses a Snappy block payload and decodes a Remote Write 2.0 `Request` via `prost`.
pub fn decode_remote_write2(compressed: &[u8]) -> Result<proto::v2::Request, String> {
    let raw = snap::raw::Decoder::new()
        .decompress_vec(compressed)
        .map_err(|e| e.to_string())?;
    proto::v2::Request::decode(raw.as_slice()).map_err(|e| e.to_string())
}

pub fn summarize_remote_write2(compressed: &[u8], req: &proto::v2::Request) -> DecodeSummary {
    let raw = snap::raw::Decoder::new()
        .decompress_vec(compressed)
        .expect("valid snappy");
    let mut s = DecodeSummary {
        uncompressed_bytes: raw.len(),
        compressed_bytes: compressed.len(),
        timeseries: req.timeseries.len(),
        samples: 0,
        histograms: 0,
        exemplars: 0,
        symbols: req.symbols.len(),
        value_bits_xor: 0,
        sample_value_sum: 0.0,
    };
    for ts in &req.timeseries {
        s.samples += ts.samples.len();
        s.histograms += ts.histograms.len();
        s.exemplars += ts.exemplars.len();
        for smp in &ts.samples {
            s.value_bits_xor ^= smp.value.to_bits();
            s.sample_value_sum += smp.value;
        }
        for h in &ts.histograms {
            s.value_bits_xor ^= h.sum.to_bits();
        }
    }
    s
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde::Deserialize;
    use std::collections::HashMap;
    use std::fs;
    use std::path::Path;

    #[derive(Deserialize)]
    struct Manifest {
        varints_count: usize,
        varints_bytes: usize,
        varints_sum: u64,
        remote_write: HashMap<String, ManifestSummary>,
    }

    #[derive(Debug, Deserialize)]
    struct ManifestSummary {
        uncompressed_bytes: usize,
        compressed_bytes: usize,
        timeseries: usize,
        samples: usize,
        histograms: usize,
        exemplars: usize,
        symbols: usize,
        value_bits_xor: u64,
        sample_value_sum: f64,
    }

    #[test]
    fn decoders_match_go_manifest() {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata");
        let manifest: Manifest =
            serde_json::from_str(&fs::read_to_string(root.join("manifest.json")).unwrap()).unwrap();

        let raw_varints = fs::read(root.join("varints_2000.bin")).unwrap();
        assert_eq!(raw_varints.len(), manifest.varints_bytes);
        let mut dst = vec![0u64; manifest.varints_count];
        let count = decode_varints(&raw_varints, &mut dst).unwrap();
        assert_eq!(count, manifest.varints_count);
        let sum: u64 = dst[..count].iter().copied().sum();
        assert_eq!(sum, manifest.varints_sum);

        for samples in [200usize, 2000usize] {
            let v2_name = format!("rw2_{samples}.snappy");
            let v2_bytes = fs::read(root.join(&v2_name)).unwrap();
            let v2_req = decode_remote_write2(&v2_bytes).unwrap();
            let got_v2 = summarize_remote_write2(&v2_bytes, &v2_req);
            let want_v2 = &manifest.remote_write[&v2_name];
            assert_eq!(got_v2.uncompressed_bytes, want_v2.uncompressed_bytes);
            assert_eq!(got_v2.compressed_bytes, want_v2.compressed_bytes);
            assert_eq!(got_v2.timeseries, want_v2.timeseries);
            assert_eq!(got_v2.samples, want_v2.samples);
            assert_eq!(got_v2.histograms, want_v2.histograms);
            assert_eq!(got_v2.exemplars, want_v2.exemplars);
            assert_eq!(got_v2.symbols, want_v2.symbols);
            assert_eq!(got_v2.value_bits_xor, want_v2.value_bits_xor);
            assert!((got_v2.sample_value_sum - want_v2.sample_value_sum).abs() < 1e-3);
        }
    }
}
