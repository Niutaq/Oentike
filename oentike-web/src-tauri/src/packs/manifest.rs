use ed25519_dalek::{Signature, VerifyingKey};
use serde::{Deserialize, Serialize};
use std::collections::HashSet;

use super::{Error, Result};

pub const DOMAIN: &[u8] = b"OENTIKE-PACK-MANIFEST-V1\0";
pub const MAX_MANIFEST: usize = 1 << 20;
pub const MAX_FILES: usize = 4096;
const MAX_FILE: u64 = 8 << 30;
const MAX_TOTAL: u64 = 32 << 30;

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Manifest {
    pub schema_version: u64,
    pub area_id: String,
    pub release: u64,
    pub created_at: String,
    pub files: Vec<Resource>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Resource {
    pub path: String,
    pub size: u64,
    pub sha256: String,
    pub format: String,
    pub source: String,
    pub license: String,
}

pub fn verify(raw: &[u8], signature: &[u8], key: &[u8; 32]) -> Result<Manifest> {
    if raw.is_empty() || raw.len() > MAX_MANIFEST || signature.len() != 64 {
        return Err(Error::Invalid("manifest or signature length"));
    }
    let key = VerifyingKey::from_bytes(key).map_err(|_| Error::Invalid("public key"))?;
    let signature = Signature::from_slice(signature).map_err(|_| Error::Invalid("signature"))?;
    let mut message = Vec::with_capacity(DOMAIN.len() + raw.len());
    message.extend_from_slice(DOMAIN);
    message.extend_from_slice(raw);
    key.verify_strict(&message, &signature)
        .map_err(|_| Error::Invalid("manifest signature"))?;
    parse(raw)
}

pub(super) fn parse(raw: &[u8]) -> Result<Manifest> {
    if raw.is_empty() || raw.len() > MAX_MANIFEST {
        return Err(Error::Invalid("manifest length"));
    }
    // Serde-derived structs also accept positional arrays. V1 only permits
    // objects, with a single array at the root's files field. Scan containers
    // outside strings first; serde performs the complete syntax/type check.
    let (mut depth, mut quoted, mut escaped) = (0usize, false, false);
    for &c in raw {
        if quoted {
            if escaped {
                escaped = false;
            } else if c == b'\\' {
                escaped = true;
            } else if c == b'"' {
                quoted = false;
            }
            continue;
        }
        match c {
            b'"' => quoted = true,
            b'{' if depth == 0 || depth == 2 => depth += 1,
            b'[' if depth == 1 => depth += 1,
            b'{' | b'[' => return Err(Error::Invalid("JSON container outside schema")),
            b'}' | b']' => depth = depth.checked_sub(1).ok_or(Error::Invalid("JSON nesting"))?,
            _ => (),
        }
    }
    // Direct typed deserialization rejects duplicate/missing fields, null,
    // unknown/case-aliased keys, floats, exponents and trailing JSON.
    // Do not deserialize through Value, which would discard duplicate keys.
    let m: Manifest = serde_json::from_slice(raw)?;
    m.validate()?;
    Ok(m)
}

impl Manifest {
    fn validate(&self) -> Result<()> {
        if self.schema_version != 1
            || !identifier(&self.area_id)
            || self.release == 0
            || self.release > (1u64 << 53) - 1
            || !utc_seconds(&self.created_at)
            || self.files.is_empty()
            || self.files.len() > MAX_FILES
        {
            return Err(Error::Invalid("manifest identity, time or file count"));
        }
        let mut paths = HashSet::with_capacity(self.files.len());
        let mut total = 0u64;
        for (i, f) in self.files.iter().enumerate() {
            if !portable_path(&f.path) || i > 0 && self.files[i - 1].path >= f.path {
                return Err(Error::Invalid("unsafe, duplicate or unsorted path"));
            }
            paths.insert(f.path.as_str());
            if f.size > MAX_FILE || f.size > MAX_TOTAL - total {
                return Err(Error::Invalid("payload size limit"));
            }
            total += f.size;
            decode_hex::<32>(&f.sha256)?;
            if !ascii(&f.format, 64) || !ascii(&f.source, 2048) || !ascii(&f.license, 256) {
                return Err(Error::Invalid("resource metadata"));
            }
        }
        for f in &self.files {
            for (i, _) in f.path.match_indices('/') {
                if paths.contains(&f.path[..i]) {
                    return Err(Error::Invalid("file/directory collision"));
                }
            }
        }
        Ok(())
    }

    pub fn total_bytes(&self) -> u64 {
        self.files.iter().map(|f| f.size).sum()
    }
}

fn ascii(s: &str, max: usize) -> bool {
    !s.is_empty() && s.len() <= max && s.bytes().all(|c| (32..=126).contains(&c))
}

pub fn identifier(s: &str) -> bool {
    !s.is_empty()
        && s.len() <= 64
        && !s.starts_with('-')
        && !s.ends_with('-')
        && s.bytes()
            .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == b'-')
}

fn portable_path(s: &str) -> bool {
    if s.is_empty()
        || s.len() > 240
        || s.split('/').count() > 8
        || s == "manifest.json"
        || s == "manifest.sig"
    {
        return false;
    }
    s.split('/').all(|part| {
        if part.is_empty()
            || part.len() > 64
            || part.starts_with('.')
            || part.ends_with('.')
            || !part
                .bytes()
                .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || b"-_.".contains(&c))
        {
            return false;
        }
        let stem = part.split('.').next().unwrap();
        !matches!(stem, "con" | "prn" | "aux" | "nul")
            && !(stem.len() == 4
                && (stem.starts_with("com") || stem.starts_with("lpt"))
                && stem.as_bytes()[3].is_ascii_digit())
    })
}

fn utc_seconds(s: &str) -> bool {
    let b = s.as_bytes();
    if b.len() != 20
        || b[4] != b'-'
        || b[7] != b'-'
        || b[10] != b'T'
        || b[13] != b':'
        || b[16] != b':'
        || b[19] != b'Z'
    {
        return false;
    }
    if !(0..20)
        .filter(|i| ![4, 7, 10, 13, 16, 19].contains(i))
        .all(|i| b[i].is_ascii_digit())
    {
        return false;
    }
    let number = |start: usize, end: usize| {
        b[start..end]
            .iter()
            .fold(0u32, |n, c| n * 10 + (c - b'0') as u32)
    };
    let year = number(0, 4);
    let leap = year % 4 == 0 && (year % 100 != 0 || year % 400 == 0);
    let days = match number(5, 7) {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 => {
            if leap {
                29
            } else {
                28
            }
        }
        _ => return false,
    };
    (1..=days).contains(&number(8, 10))
        && number(11, 13) < 24
        && number(14, 16) < 60
        && number(17, 19) < 60
}

pub fn decode_hex<const N: usize>(s: &str) -> Result<[u8; N]> {
    if s.len() != N * 2
        || !s
            .bytes()
            .all(|c| c.is_ascii_digit() || (b'a'..=b'f').contains(&c))
    {
        return Err(Error::Invalid("lowercase hex"));
    }
    let mut bytes = [0; N];
    for (i, chunk) in s.as_bytes().chunks_exact(2).enumerate() {
        let digit = |c: u8| if c <= b'9' { c - b'0' } else { c - b'a' + 10 };
        bytes[i] = digit(chunk[0]) * 16 + digit(chunk[1]);
    }
    Ok(bytes)
}
