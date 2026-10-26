/// Opens only the macOS Location Services settings pane, on explicit user action.
/// This does not grant permission, reset a denial or change system settings.
#[tauri::command]
pub async fn open_location_settings() -> Result<(), String> {
    #[cfg(target_os = "macos")]
    {
        let status = std::process::Command::new("/usr/bin/open")
            .arg("x-apple.systempreferences:com.apple.preference.security?Privacy_LocationServices")
            .status()
            .map_err(|error| format!("open location settings: {error}"))?;
        if status.success() {
            Ok(())
        } else {
            Err("could not open location settings".into())
        }
    }
    #[cfg(not(target_os = "macos"))]
    Err("location settings shortcut is available on macOS only".into())
}
