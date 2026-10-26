//! Unix importer. Store directories are private to this app and published files
//! are immutable. Untrusted source files are streamed, never memory mapped.
use memmap2::MmapOptions;
use rustix::fs::{flock, fstatvfs, open, openat, Dir, FlockOperation, Mode, OFlags};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs::{self, File, OpenOptions},
    io::{Read, Write},
    os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt, PermissionsExt},
    path::{Path, PathBuf},
};
use tempfile::{Builder, NamedTempFile};

use super::{
    manifest::{self, Manifest, Resource, MAX_MANIFEST},
    Error, ImportResult, PackInfo, Result,
};

const READ_FLAGS: OFlags = OFlags::RDONLY
    .union(OFlags::CLOEXEC)
    .union(OFlags::NOFOLLOW)
    .union(OFlags::NONBLOCK);

pub(super) struct Store {
    root: PathBuf,
    key: [u8; 32],
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Active {
    directory: String,
    release: u64,
    manifest_sha256: String,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub(super) enum Phase {
    Copied,
    Published,
    BeforeCommit,
}

impl Store {
    pub fn open(root: PathBuf, key: [u8; 32]) -> Result<Self> {
        if let Some(parent) = root.parent() {
            create_parents(parent)?;
        }
        private_dir(&root)?;
        Ok(Self { root, key })
    }

    fn lock(&self) -> Result<File> {
        let file = OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .truncate(false)
            .mode(0o600)
            .custom_flags((OFlags::NOFOLLOW | OFlags::NONBLOCK).bits() as i32)
            .open(self.root.join("import.lock"))?;
        regular(&file)?;
        flock(&file, FlockOperation::NonBlockingLockExclusive)
            .map_err(|_| Error::Invalid("another pack import is running"))?;
        Ok(file) // RAII unlock; never unlink/replace the lock inode.
    }

    pub fn import(
        &self,
        area: &str,
        payload: &Path,
        raw: &Path,
        signature: &Path,
    ) -> Result<ImportResult> {
        self.import_inner(area, payload, raw, signature, |_| Ok(()))
    }

    pub(super) fn import_inner(
        &self,
        area: &str,
        payload: &Path,
        raw_path: &Path,
        sig_path: &Path,
        checkpoint: impl Fn(Phase) -> Result<()>,
    ) -> Result<ImportResult> {
        if !manifest::identifier(area) {
            return Err(Error::Invalid("area id"));
        }
        let _lock = self.lock()?;
        let raw = read_small(raw_path, MAX_MANIFEST)?;
        let signature = read_small(sig_path, 64)?;
        let m = manifest::verify(&raw, &signature, &self.key)?;
        if m.area_id != area {
            return Err(Error::Invalid("pack belongs to another area"));
        }
        if let Some((_, previous)) = self.current(area)? {
            if m.release <= previous.release {
                return Err(Error::Invalid("release must be newer than the active pack"));
            }
        }
        let source = open_dir(payload)?;
        inventory(&source, &m)?;
        let available = fstatvfs(File::open(&self.root)?)?;
        let free = available.f_bavail.saturating_mul(available.f_frsize);
        if m.total_bytes() + raw.len() as u64 + (16 << 20) > free {
            return Err(Error::Invalid("insufficient free disk space for pack"));
        }
        let area_dir = self.root.join(area);
        private_dir(&area_dir)?;
        let releases = area_dir.join("releases");
        private_dir(&releases)?;
        // Staging and final directories share a filesystem. TempDir removes only
        // this attempt's staging tree on any pre-publication error.
        let staging = Builder::new()
            .prefix(".stage-")
            .permissions(fs::Permissions::from_mode(0o700))
            .tempdir_in(&releases)?;
        let target_payload = staging.path().join("payload");
        private_dir(&target_payload)?;
        let mut buffer = [0u8; 64 << 10];
        for resource in &m.files {
            copy_resource(&source, &target_payload, resource, &mut buffer)?;
        }
        // Detect additions/removals during copying as well. Content authenticity
        // comes from the verified private copies, not mutable source paths.
        inventory(&source, &m)?;
        checkpoint(Phase::Copied)?;
        write_new(&staging.path().join("manifest.json"), &raw)?;
        write_new(&staging.path().join("manifest.sig"), &signature)?;
        sync_tree_dirs(&target_payload, &m)?;
        File::open(staging.path())?.sync_all()?;

        let suffix = staging
            .path()
            .file_name()
            .and_then(|s| s.to_str())
            .and_then(|s| s.strip_prefix(".stage-"))
            .ok_or(Error::Invalid("staging name"))?;
        let directory = format!("pack-{suffix}");
        let published = releases.join(&directory);
        if published.try_exists()? {
            return Err(Error::Invalid("release directory collision"));
        }
        fs::rename(staging.path(), &published)?;
        // A failure below can leave a complete orphan release. It never changes
        // active.json; a later import can retry under a fresh directory name.
        File::open(&releases)?.sync_all()?;
        checkpoint(Phase::Published)?;

        let state = Active {
            directory,
            release: m.release,
            manifest_sha256: hex(&Sha256::digest(&raw)),
        };
        let mut pointer = NamedTempFile::new_in(&area_dir)?;
        pointer.write_all(&serde_json::to_vec(&state)?)?;
        pointer.as_file().sync_all()?;
        File::open(&area_dir)?.sync_all()?;
        File::open(&self.root)?.sync_all()?;
        checkpoint(Phase::BeforeCommit)?;
        fs::rename(pointer.path(), area_dir.join("active.json"))?;
        // Commit point. Never return Err after this rename: the new pack is now
        // active. Report an fsync failure explicitly as unconfirmed durability.
        let durability_confirmed = File::open(&area_dir).and_then(|d| d.sync_all()).is_ok();
        Ok(ImportResult {
            pack: PackInfo::from(&m),
            durability_confirmed,
        })
    }

    pub fn active(&self, area: &str) -> Result<Option<PackInfo>> {
        let Some((directory, m)) = self.current(area)? else {
            return Ok(None);
        };
        let payload = open_dir(&directory.join("payload"))?;
        inventory(&payload, &m)?;
        for f in &m.files {
            let file = open_resource(&payload, &f.path)?;
            verify_private_file(&file, f)?;
        }
        Ok(Some(PackInfo::from(&m)))
    }

    fn current(&self, area: &str) -> Result<Option<(PathBuf, Manifest)>> {
        if !manifest::identifier(area) {
            return Err(Error::Invalid("area id"));
        }
        let area_dir = self.root.join(area);
        if !area_dir.try_exists()? {
            return Ok(None);
        }
        check_private_dir(&area_dir)?;
        let raw = match read_small(&area_dir.join("active.json"), 2048) {
            Ok(raw) => raw,
            Err(Error::Io(err)) if err.kind() == std::io::ErrorKind::NotFound => return Ok(None),
            Err(err) => return Err(err),
        };
        let state: Active = serde_json::from_slice(&raw)?;
        if !state.directory.starts_with("pack-")
            || state.directory.len() > 64
            || !state.directory[5..]
                .bytes()
                .all(|c| c.is_ascii_alphanumeric())
            || state.directory.len() <= 5
        {
            return Err(Error::Invalid("active directory"));
        }
        let releases = area_dir.join("releases");
        check_private_dir(&releases)?;
        let directory = releases.join(&state.directory);
        check_private_dir(&directory)?;
        let raw = read_small(&directory.join("manifest.json"), MAX_MANIFEST)?;
        let sig = read_small(&directory.join("manifest.sig"), 64)?;
        let m = manifest::verify(&raw, &sig, &self.key)?;
        if m.area_id != area
            || m.release != state.release
            || manifest::decode_hex::<32>(&state.manifest_sha256)?
                != <[u8; 32]>::from(Sha256::digest(&raw))
        {
            return Err(Error::Invalid(
                "active pointer does not match signed manifest",
            ));
        }
        Ok(Some((directory, m)))
    }
}

fn private_dir(path: &Path) -> Result<()> {
    match fs::DirBuilder::new().mode(0o700).create(path) {
        Ok(()) => {
            if let Some(parent) = path.parent() {
                File::open(parent)?.sync_all()?;
            }
        }
        Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => (),
        Err(e) => return Err(e.into()),
    }
    check_private_dir(path)
}

// The app data path can be new on first launch. Persist newly created parent
// links as well, before reporting a durable activation under that path.
fn create_parents(path: &Path) -> Result<()> {
    if path.is_dir() {
        return Ok(());
    }
    if let Some(parent) = path.parent() {
        create_parents(parent)?;
    }
    match fs::DirBuilder::new().mode(0o700).create(path) {
        Ok(()) => (),
        Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists && path.is_dir() => (),
        Err(e) => return Err(e.into()),
    }
    if let Some(parent) = path.parent() {
        File::open(parent)?.sync_all()?;
    }
    Ok(())
}

fn check_private_dir(path: &Path) -> Result<()> {
    let meta = fs::symlink_metadata(path)?;
    if !meta.is_dir() || meta.permissions().mode() & 0o077 != 0 {
        return Err(Error::Invalid("store must use private, real directories"));
    }
    Ok(())
}

fn open_dir(path: &Path) -> Result<File> {
    Ok(File::from(open(
        path,
        READ_FLAGS | OFlags::DIRECTORY,
        Mode::empty(),
    )?))
}

fn regular(file: &File) -> Result<fs::Metadata> {
    let meta = file.metadata()?;
    if !meta.is_file() || meta.nlink() != 1 {
        return Err(Error::Invalid("special file or hardlink"));
    }
    Ok(meta)
}

fn read_small(path: &Path, max: usize) -> Result<Vec<u8>> {
    let file = File::from(open(path, READ_FLAGS, Mode::empty())?);
    if regular(&file)?.len() > max as u64 {
        return Err(Error::Invalid("metadata size limit"));
    }
    let mut raw = Vec::new();
    file.take(max as u64 + 1).read_to_end(&mut raw)?;
    if raw.len() > max {
        return Err(Error::Invalid("metadata size limit"));
    }
    Ok(raw)
}

// Walk each component relative to an already-open directory, refusing symlinks
// at every level. Renaming a source directory cannot redirect traversal outside it.
fn open_resource(root: &File, path: &str) -> Result<File> {
    let mut dir = root.try_clone()?;
    let mut parts = path.split('/').peekable();
    while let Some(part) = parts.next() {
        let flags = if parts.peek().is_some() {
            READ_FLAGS | OFlags::DIRECTORY
        } else {
            READ_FLAGS
        };
        dir = File::from(openat(&dir, part, flags, Mode::empty())?);
    }
    regular(&dir)?;
    Ok(dir)
}

fn inventory(root: &File, m: &Manifest) -> Result<()> {
    let mut files: BTreeMap<&str, u64> =
        m.files.iter().map(|f| (f.path.as_str(), f.size)).collect();
    let mut dirs = BTreeSet::new();
    for f in &m.files {
        for (i, _) in f.path.match_indices('/') {
            dirs.insert(&f.path[..i]);
        }
    }
    fn walk(
        dir: &File,
        prefix: &str,
        files: &mut BTreeMap<&str, u64>,
        dirs: &BTreeSet<&str>,
    ) -> Result<()> {
        for entry in Dir::read_from(dir)? {
            let entry = entry?;
            let name = entry
                .file_name()
                .to_str()
                .map_err(|_| Error::Invalid("non-ASCII filename"))?;
            if name == "." || name == ".." {
                continue;
            }
            let path = if prefix.is_empty() {
                name.to_owned()
            } else {
                format!("{prefix}/{name}")
            };
            if dirs.contains(path.as_str()) {
                let child = File::from(openat(
                    dir,
                    name,
                    READ_FLAGS | OFlags::DIRECTORY,
                    Mode::empty(),
                )?);
                walk(&child, &path, files, dirs)?;
            } else if let Some(size) = files.remove(path.as_str()) {
                let child = File::from(openat(dir, name, READ_FLAGS, Mode::empty())?);
                if regular(&child)?.len() != size {
                    return Err(Error::Invalid("payload file size"));
                }
            } else {
                return Err(Error::Invalid("unlisted payload entry"));
            }
        }
        Ok(())
    }
    walk(root, "", &mut files, &dirs)?;
    if !files.is_empty() {
        return Err(Error::Invalid("missing payload file"));
    }
    Ok(())
}

fn copy_resource(source: &File, target: &Path, f: &Resource, buffer: &mut [u8]) -> Result<()> {
    let mut input = open_resource(source, &f.path)?;
    if regular(&input)?.len() != f.size {
        return Err(Error::Invalid("source size changed"));
    }
    let output_path = target.join(&f.path);
    // All components were validated and target is this attempt's private tree.
    fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(output_path.parent().unwrap())?;
    let mut output = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(&output_path)?;
    let mut remaining = f.size;
    while remaining != 0 {
        let count = remaining.min(buffer.len() as u64) as usize;
        input.read_exact(&mut buffer[..count])?;
        output.write_all(&buffer[..count])?;
        remaining -= count as u64;
    }
    if input.read(&mut buffer[..1])? != 0 {
        return Err(Error::Invalid("source grew during import"));
    }
    output.sync_all()?;
    output.set_permissions(fs::Permissions::from_mode(0o400))?;
    drop(output); // Close the only writer before mapping our private copy.
    verify_private_file(&File::open(output_path)?, f)
}

fn verify_private_file(file: &File, f: &Resource) -> Result<()> {
    if regular(file)?.len() != f.size || f.size > isize::MAX as u64 {
        return Err(Error::Invalid("private file size"));
    }
    let digest: [u8; 32] = if f.size == 0 {
        Sha256::digest([]).into()
    } else {
        // SAFETY: only called on private staging files after closing the writer,
        // or published store files. This module never mutates published files,
        // never maps untrusted source paths, and keeps releases across updates.
        // The app store must not be modified by other same-user processes.
        let map = unsafe { MmapOptions::new().map(file)? };
        Sha256::digest(&map[..]).into()
    };
    if digest != manifest::decode_hex::<32>(&f.sha256)? {
        return Err(Error::Invalid("payload SHA-256 mismatch"));
    }
    Ok(())
}

fn write_new(path: &Path, raw: &[u8]) -> Result<()> {
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o400)
        .open(path)?;
    file.write_all(raw)?;
    file.sync_all()?;
    Ok(())
}

fn sync_tree_dirs(root: &Path, m: &Manifest) -> Result<()> {
    let mut dirs = BTreeSet::new();
    for f in &m.files {
        for (i, _) in f.path.match_indices('/') {
            dirs.insert(&f.path[..i]);
        }
    }
    for dir in dirs.iter().rev() {
        File::open(root.join(dir))?.sync_all()?;
    }
    File::open(root)?.sync_all()?;
    Ok(())
}

fn hex(bytes: &[u8]) -> String {
    use std::fmt::Write;
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        write!(out, "{b:02x}").unwrap();
    }
    out
}
