use super::{manifest, Error};
use ed25519_dalek::{Signer, SigningKey};
use serde::Deserialize;

#[derive(Deserialize)]
struct Vector {
    seed_hex: String,
    public_key_hex: String,
    manifest: String,
    signature_hex: String,
    payload_hex: String,
}
fn vector() -> Vector {
    serde_json::from_str(include_str!(
        "../../../../oentike-api/internal/pack/testdata/vector.json"
    ))
    .unwrap()
}
fn sign(raw: &[u8]) -> Vec<u8> {
    let key = SigningKey::from_bytes(&manifest::decode_hex::<32>(&vector().seed_hex).unwrap());
    let mut message = manifest::DOMAIN.to_vec();
    message.extend_from_slice(raw);
    key.sign(&message).to_bytes().to_vec()
}

#[test]
fn go_wire_vector_and_signature_binding() {
    let v = vector();
    let key = manifest::decode_hex::<32>(&v.public_key_hex).unwrap();
    let sig = manifest::decode_hex::<64>(&v.signature_hex).unwrap();
    assert_eq!(sign(v.manifest.as_bytes()), sig);
    let m = manifest::verify(v.manifest.as_bytes(), &sig, &key).unwrap();
    assert_eq!(serde_json::to_string(&m).unwrap(), v.manifest);
    assert_eq!(m.files[0].size, 6);
    for input in [
        v.manifest.replace("nadl-05-31", "nadl-05-32"),
        format!("{}\n", v.manifest),
    ] {
        assert!(manifest::verify(input.as_bytes(), &sig, &key).is_err());
    }
    let mut wrong = sig;
    wrong[0] ^= 1;
    assert!(manifest::verify(v.manifest.as_bytes(), &wrong, &key).is_err());
    assert!(manifest::verify(v.manifest.as_bytes(), &sig[..63], &key).is_err());
    assert!(manifest::verify(v.manifest.as_bytes(), &sig, &[1; 32]).is_err());
    let no_domain = SigningKey::from_bytes(&[0; 32]).sign(v.manifest.as_bytes());
    assert!(manifest::verify(v.manifest.as_bytes(), &no_domain.to_bytes(), &key).is_err());
    assert!(manifest::verify(&vec![b' '; manifest::MAX_MANIFEST + 1], &sig, &key).is_err());
}

#[test]
fn correctly_signed_malformed_manifests_are_rejected() {
    let v = vector();
    let key = manifest::decode_hex::<32>(&v.public_key_hex).unwrap();
    let substitutions = [
        ("\"release\":1", "\"release\":1,\"release\":2"),
        ("\"release\":1", "\"release\":1,\"rele\\u0061se\":2"),
        ("\"release\":1", "\"Release\":1"),
        ("\"release\":1", "\"release\":0"),
        ("\"release\":1", "\"release\":9007199254740992"),
        ("\"schema_version\":1", "\"schema_version\":2"),
        ("\"size\":6", "\"size\":6,\"size\":7"),
        ("\"size\":6", "\"size\":null"),
        ("\"size\":6", "\"size\":6.0"),
        ("\"size\":6", "\"size\":6e0"),
        ("\"size\":6", "\"size\":-1"),
        ("\"size\":6", "\"size\":-0"),
        ("\"size\":6", "\"size\":8589934593"),
        ("\"size\":6,", ""),
        ("\"size\":6", "\"size\":6,\"extra\":1"),
        ("synthetic:test", "\\ud800"),
        ("synthetic:test", "żółć"),
        ("data/snapshot.json", "../snapshot.json"),
        ("data/snapshot.json", "a/b/../../outside"),
        ("data/snapshot.json", "Data/a"),
        ("data/snapshot.json", "nul.txt"),
        ("data/snapshot.json", "a//b"),
        ("data/snapshot.json", "manifest.sig"),
        ("2026-10-02T12:00:00Z", "2026-02-29T12:00:00Z"),
        ("2026-10-02T12:00:00Z", "2026-10-02T12:00:00.1Z"),
    ];
    for (from, to) in substitutions {
        let raw = v.manifest.replace(from, to);
        assert_ne!(raw, v.manifest);
        assert!(
            manifest::verify(raw.as_bytes(), &sign(raw.as_bytes()), &key).is_err(),
            "accepted {to}"
        );
    }
    let pretty = serde_json::to_string_pretty(
        &serde_json::from_str::<serde_json::Value>(&v.manifest).unwrap(),
    )
    .unwrap();
    assert!(manifest::verify(pretty.as_bytes(), &sign(pretty.as_bytes()), &key).is_ok());
    let positional = r#"[1,"nadl-05-31",1,"2026-10-02T12:00:00Z",[]]"#;
    assert!(manifest::parse(positional.as_bytes()).is_err());
    let m: manifest::Manifest = manifest::parse(v.manifest.as_bytes()).unwrap();
    let f = &m.files[0];
    let array = serde_json::json!([f.path, f.size, f.sha256, f.format, f.source, f.license]);
    let raw = v
        .manifest
        .replace(&serde_json::to_string(f).unwrap(), &array.to_string());
    assert!(manifest::parse(raw.as_bytes()).is_err());
}

#[test]
fn trust_configuration_fails_closed() {
    assert!(super::configured_key(None).is_err());
    assert!(super::configured_key(Some("bad")).is_err());
    assert!(super::configured_key(Some(&vector().public_key_hex)).is_err());
    assert!(super::configured_key(Some(&"0".repeat(64))).is_err());
}

#[test]
fn parser_mutations_do_not_panic() {
    let seed = vector().manifest.into_bytes();
    for len in 0..seed.len() {
        let _ = manifest::parse(&seed[..len]);
    }
    for i in 0..seed.len() {
        for byte in [0, b'"', b'\\', b'[', b'{', b'}', b']', 0xff] {
            let mut data = seed.clone();
            data[i] = byte;
            let _ = manifest::parse(&data);
        }
    }
    assert!(manifest::parse(&vec![b'['; 10000]).is_err());
}

#[cfg(unix)]
mod importer {
    use super::*;
    use crate::packs::store::{Phase, Store};
    use std::{fs, os::unix::fs::symlink, path::PathBuf};

    struct Fixture {
        _dir: tempfile::TempDir,
        root: PathBuf,
        payload: PathBuf,
        raw: PathBuf,
        sig: PathBuf,
    }
    impl Fixture {
        fn new() -> Self {
            let dir = tempfile::tempdir().unwrap();
            let root = dir.path().join("store");
            let payload = dir.path().join("input");
            fs::create_dir_all(payload.join("data")).unwrap();
            fs::write(
                payload.join("data/snapshot.json"),
                manifest::decode_hex::<6>(&vector().payload_hex).unwrap(),
            )
            .unwrap();
            let raw = dir.path().join("manifest.json");
            let sig = dir.path().join("manifest.sig");
            fs::write(&raw, vector().manifest).unwrap();
            fs::write(
                &sig,
                manifest::decode_hex::<64>(&vector().signature_hex).unwrap(),
            )
            .unwrap();
            Self {
                _dir: dir,
                root,
                payload,
                raw,
                sig,
            }
        }
        fn store(&self) -> Store {
            Store::open(
                self.root.clone(),
                manifest::decode_hex::<32>(&vector().public_key_hex).unwrap(),
            )
            .unwrap()
        }
        fn release(&self, n: u64) {
            let raw = vector()
                .manifest
                .replace("\"release\":1", &format!("\"release\":{n}"));
            fs::write(&self.raw, &raw).unwrap();
            fs::write(&self.sig, sign(raw.as_bytes())).unwrap();
        }
        fn import(&self) -> super::super::Result<super::super::ImportResult> {
            self.store()
                .import("nadl-05-31", &self.payload, &self.raw, &self.sig)
        }
        fn pointer(&self) -> Vec<u8> {
            fs::read(self.root.join("nadl-05-31/active.json")).unwrap()
        }
        fn active_payload(&self) -> PathBuf {
            let pointer: serde_json::Value = serde_json::from_slice(&self.pointer()).unwrap();
            self.root
                .join("nadl-05-31/releases")
                .join(pointer["directory"].as_str().unwrap())
                .join("payload/data/snapshot.json")
        }
    }

    #[test]
    fn import_restart_update_keeps_old_open_mapping() {
        let f = Fixture::new();
        assert!(f.store().active("nadl-05-31").unwrap().is_none());
        let result = f.import().unwrap();
        assert_eq!(result.pack.release, 1);
        assert!(result.durability_confirmed);
        assert_eq!(f.store().active("nadl-05-31").unwrap().unwrap().release, 1);
        let old_file = fs::File::open(f.active_payload()).unwrap();
        // SAFETY: test owns this immutable store; import only writes new files.
        let old_map = unsafe { memmap2::MmapOptions::new().map(&old_file).unwrap() };
        f.release(2);
        let mut next = manifest::parse(&fs::read(&f.raw).unwrap()).unwrap();
        use sha2::{Digest, Sha256};
        next.files[0].sha256 = Sha256::digest(b"newer\n")
            .iter()
            .map(|b| format!("{b:02x}"))
            .collect();
        let next_raw = serde_json::to_vec(&next).unwrap();
        fs::write(&f.raw, &next_raw).unwrap();
        fs::write(&f.sig, sign(&next_raw)).unwrap();
        fs::write(f.payload.join("data/snapshot.json"), b"newer\n").unwrap();
        assert_eq!(f.import().unwrap().pack.release, 2);
        assert_eq!(&old_map[..], b"hello\n");
        assert_eq!(fs::read(f.active_payload()).unwrap(), b"newer\n");
        assert_eq!(f.store().active("nadl-05-31").unwrap().unwrap().release, 2);
        let active = f.pointer();
        assert!(f.import().is_err());
        f.release(1);
        assert!(f.import().is_err());
        assert_eq!(f.pointer(), active);
        assert!(f.store().active("../escape").is_err());
        assert!(f
            .store()
            .import("another-area", &f.payload, &f.raw, &f.sig)
            .is_err());
    }

    #[test]
    fn errors_before_commit_preserve_pointer_and_allow_retry() {
        for phase in [Phase::Copied, Phase::Published, Phase::BeforeCommit] {
            let f = Fixture::new();
            f.import().unwrap();
            let before = f.pointer();
            f.release(2);
            let err = f
                .store()
                .import_inner("nadl-05-31", &f.payload, &f.raw, &f.sig, |at| {
                    if at == phase {
                        Err(Error::Io(std::io::Error::new(
                            std::io::ErrorKind::StorageFull,
                            "injected disk failure",
                        )))
                    } else {
                        Ok(())
                    }
                });
            assert!(err.is_err());
            assert_eq!(f.pointer(), before);
            assert_eq!(f.store().active("nadl-05-31").unwrap().unwrap().release, 1);
            assert_eq!(f.import().unwrap().pack.release, 2);
        }
    }

    #[test]
    fn invalid_payloads_never_replace_active() {
        for case in [
            "hash",
            "missing",
            "extra",
            "extra-dir",
            "symlink",
            "parent-symlink",
            "hardlink",
            "fifo",
            "signature",
            "truncated",
        ] {
            let f = Fixture::new();
            f.import().unwrap();
            let before = f.pointer();
            f.release(2);
            let path = f.payload.join("data/snapshot.json");
            match case {
                "hash" => fs::write(&path, b"HELLO\n").unwrap(),
                "missing" => fs::remove_file(&path).unwrap(),
                "extra" => fs::write(f.payload.join("extra"), b"x").unwrap(),
                "extra-dir" => fs::create_dir(f.payload.join("empty")).unwrap(),
                "symlink" => {
                    fs::remove_file(&path).unwrap();
                    symlink(&f.raw, &path).unwrap();
                }
                "parent-symlink" => {
                    fs::remove_file(&path).unwrap();
                    fs::remove_dir(path.parent().unwrap()).unwrap();
                    symlink(f._dir.path(), path.parent().unwrap()).unwrap();
                }
                "hardlink" => fs::hard_link(&path, f._dir.path().join("other-link")).unwrap(),
                "fifo" => {
                    fs::remove_file(&path).unwrap();
                    assert!(std::process::Command::new("mkfifo")
                        .arg(&path)
                        .status()
                        .unwrap()
                        .success());
                }
                "signature" => fs::write(&f.sig, [0u8; 64]).unwrap(),
                "truncated" => fs::write(&path, b"x").unwrap(),
                _ => unreachable!(),
            }
            assert!(f.import().is_err(), "accepted {case}");
            assert_eq!(f.pointer(), before);
            assert_eq!(f.store().active("nadl-05-31").unwrap().unwrap().release, 1);
        }
    }

    #[test]
    fn source_changes_after_copy_do_not_change_installed_bytes() {
        let f = Fixture::new();
        f.store()
            .import_inner("nadl-05-31", &f.payload, &f.raw, &f.sig, |at| {
                if at == Phase::Copied {
                    fs::write(f.payload.join("data/snapshot.json"), b"HELLO\n")?;
                }
                Ok(())
            })
            .unwrap();
        assert_eq!(fs::read(f.active_payload()).unwrap(), b"hello\n");
        assert!(f.store().active("nadl-05-31").unwrap().is_some());
    }

    #[test]
    fn concurrent_import_is_rejected_and_corrupt_pointer_fails_closed() {
        let f = Fixture::new();
        f.import().unwrap();
        f.release(2);
        let lock = fs::OpenOptions::new()
            .read(true)
            .write(true)
            .open(f.root.join("import.lock"))
            .unwrap();
        rustix::fs::flock(&lock, rustix::fs::FlockOperation::LockExclusive).unwrap();
        assert!(f.import().is_err());
        drop(lock);
        let before = f.pointer();
        let pointer = f.root.join("nadl-05-31/active.json");
        fs::write(&pointer, b"{}").unwrap();
        assert!(f.store().active("nadl-05-31").is_err());
        assert!(f.import().is_err());
        fs::write(pointer, before).unwrap();
        assert!(f.import().is_ok());
    }

    #[test]
    fn active_detects_changed_payload_and_ignores_abandoned_staging() {
        let f = Fixture::new();
        f.import().unwrap();
        fs::create_dir(f.root.join("nadl-05-31/releases/.stage-crashed")).unwrap();
        assert!(f.store().active("nadl-05-31").unwrap().is_some());
        let file = f.active_payload();
        use std::os::unix::fs::PermissionsExt;
        fs::set_permissions(&file, fs::Permissions::from_mode(0o600)).unwrap();
        fs::write(file, b"HELLO\n").unwrap();
        assert!(f.store().active("nadl-05-31").is_err());
    }

    #[test]
    fn empty_resource_is_supported() {
        let f = Fixture::new();
        fs::write(f.payload.join("data/snapshot.json"), b"").unwrap();
        let raw = vector()
            .manifest
            .replace("\"size\":6", "\"size\":0")
            .replace(
                "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
            );
        fs::write(&f.raw, &raw).unwrap();
        fs::write(&f.sig, sign(raw.as_bytes())).unwrap();
        f.import().unwrap();
        assert_eq!(
            f.store().active("nadl-05-31").unwrap().unwrap().total_bytes,
            0
        );
    }
}
