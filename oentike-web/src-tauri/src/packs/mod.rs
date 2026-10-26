//! Signed inventory verification and immutable local pack installation.
mod manifest;
#[cfg(unix)]
mod store;
#[cfg(test)]
mod tests;

use serde::Serialize;
use std::{fmt, path::PathBuf};
use tauri::Manager;

type Result<T> = std::result::Result<T, Error>;

#[derive(Debug)]
enum Error {
    Invalid(&'static str),
    Io(std::io::Error),
    Json(serde_json::Error),
}
impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Invalid(message) => write!(f, "offline pack: {message}"),
            Self::Io(err) => write!(f, "offline pack I/O: {err}"),
            Self::Json(err) => write!(f, "offline pack JSON: {err}"),
        }
    }
}
impl From<std::io::Error> for Error {
    fn from(err: std::io::Error) -> Self {
        Self::Io(err)
    }
}
impl From<serde_json::Error> for Error {
    fn from(err: serde_json::Error) -> Self {
        Self::Json(err)
    }
}
#[cfg(unix)]
impl From<rustix::io::Errno> for Error {
    fn from(err: rustix::io::Errno) -> Self {
        Self::Io(err.into())
    }
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct PackInfo {
    area_id: String,
    release: u64,
    created_at: String,
    file_count: usize,
    total_bytes: u64,
}
impl From<&manifest::Manifest> for PackInfo {
    fn from(m: &manifest::Manifest) -> Self {
        Self {
            area_id: m.area_id.clone(),
            release: m.release,
            created_at: m.created_at.clone(),
            file_count: m.files.len(),
            total_bytes: m.total_bytes(),
        }
    }
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ImportResult {
    pack: PackInfo,
    /// False means activation succeeded, but the final directory fsync failed.
    durability_confirmed: bool,
}

fn embedded_key() -> Result<[u8; 32]> {
    configured_key(option_env!("OENTIKE_PACK_PUBLIC_KEY_HEX"))
}

fn configured_key(encoded: Option<&str>) -> Result<[u8; 32]> {
    let encoded = encoded.ok_or(Error::Invalid("no public key configured at build time"))?;
    let key = manifest::decode_hex(encoded)?;
    // The cross-language fixture's private seed is public. Never trust it in an app build.
    if encoded == "3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29" {
        return Err(Error::Invalid(
            "fixture key cannot be a production trust root",
        ));
    }
    let verifying =
        ed25519_dalek::VerifyingKey::from_bytes(&key).map_err(|_| Error::Invalid("public key"))?;
    if verifying.is_weak() {
        return Err(Error::Invalid("weak public key"));
    }
    Ok(key)
}

fn store_path(app: &tauri::AppHandle) -> std::result::Result<PathBuf, String> {
    app.path()
        .app_data_dir()
        .map(|p| p.join("offline-packs-v1"))
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn import_offline_pack(
    app: tauri::AppHandle,
    area_id: String,
    payload_dir: PathBuf,
    manifest_path: PathBuf,
    signature_path: PathBuf,
) -> std::result::Result<ImportResult, String> {
    let key = embedded_key().map_err(|e| e.to_string())?;
    let root = store_path(&app)?;
    tauri::async_runtime::spawn_blocking(move || {
        #[cfg(unix)]
        {
            store::Store::open(root, key)?.import(
                &area_id,
                &payload_dir,
                &manifest_path,
                &signature_path,
            )
        }
        #[cfg(not(unix))]
        {
            let _ = (
                root,
                key,
                area_id,
                payload_dir,
                manifest_path,
                signature_path,
            );
            Err(Error::Invalid("pack import currently requires Unix"))
        }
    })
    .await
    .map_err(|e| e.to_string())?
    .map_err(|e: Error| e.to_string())
}

#[tauri::command]
pub async fn get_offline_pack(
    app: tauri::AppHandle,
    area_id: String,
) -> std::result::Result<Option<PackInfo>, String> {
    let key = embedded_key().map_err(|e| e.to_string())?;
    let root = store_path(&app)?;
    tauri::async_runtime::spawn_blocking(move || {
        #[cfg(unix)]
        {
            store::Store::open(root, key)?.active(&area_id)
        }
        #[cfg(not(unix))]
        {
            let _ = (root, key, area_id);
            Err(Error::Invalid("pack import currently requires Unix"))
        }
    })
    .await
    .map_err(|e| e.to_string())?
    .map_err(|e: Error| e.to_string())
}
