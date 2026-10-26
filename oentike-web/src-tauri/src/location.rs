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
use serde::Serialize;

/// Error codes shared with the expedition UI:
/// 1 = permission denied, 2 = unavailable, 3 = timeout, 4 = untrusted host.
#[derive(Debug, Serialize)]
pub struct LocationFix {
    latitude: f64,
    longitude: f64,
    accuracy: f64,
    timestamp: f64,
}

#[derive(Debug, Serialize)]
pub struct LocationError {
    code: u8,
    message: String,
}

impl LocationError {
    fn new(code: u8, message: &str) -> Self {
        Self {
            code,
            message: message.into(),
        }
    }
}

#[derive(Debug, Serialize)]
pub struct LocationHostStatus {
    /// True when the process is running from a macOS `.app` bundle.
    trusted: bool,
    /// `bundle` or `binary` — never includes path components that look like home dirs.
    kind: &'static str,
}

fn usable_fix(accuracy: f64, timestamp: f64, now: f64) -> bool {
    accuracy.is_finite()
        && accuracy >= 0.0
        && timestamp.is_finite()
        && timestamp > 0.0
        && (now - timestamp).abs() <= 30_000.0
}

/// Core Location's permission prompt is reliable only for a real `.app` bundle.
/// Naked `tauri dev` binaries keep authorization at NotDetermined and time out.
fn path_is_app_bundle(path: &std::path::Path) -> bool {
    path.to_str()
        .is_some_and(|s| s.contains(".app/Contents/MacOS/"))
}

fn current_host_trusted() -> bool {
    std::env::current_exe()
        .ok()
        .is_some_and(|path| path_is_app_bundle(&path))
}

fn normalize_purpose(purpose: &str) -> Result<&str, LocationError> {
    match purpose.trim() {
        purpose @ ("parking" | "trail") => Ok(purpose),
        _ => Err(LocationError::new(
            2,
            "Location purpose must be parking or trail",
        )),
    }
}

/// Privacy audit line: purpose and outcome only — never coordinates.
fn audit(event: &str, purpose: &str, detail: &str) {
    log::info!("[location.audit] event={event} purpose={purpose} {detail}");
}

#[tauri::command]
pub fn location_host_status() -> LocationHostStatus {
    let trusted = current_host_trusted();
    LocationHostStatus {
        trusted,
        kind: if trusted { "bundle" } else { "binary" },
    }
}

/// Each request is tied to its caller's token so a late timeout/cancel cannot
/// stop a newer request. Core Location and its delegate live on the main runloop.
/// `purpose` is required (`parking` | `trail`) for least-privilege auditing.
#[tauri::command]
pub async fn get_current_location(
    app: tauri::AppHandle,
    request_id: String,
    purpose: String,
) -> Result<LocationFix, LocationError> {
    let purpose = normalize_purpose(&purpose)?.to_string();

    #[cfg(target_os = "macos")]
    {
        if !current_host_trusted() {
            audit(
                "rejected",
                &purpose,
                "outcome=untrusted_host host=binary",
            );
            return Err(LocationError::new(
                4,
                "Location requires the Oentike.app bundle (./oentike field)",
            ));
        }

        audit("request", &purpose, "host=bundle");
        let (send, receive) = tokio::sync::oneshot::channel();
        let token = request_id.clone();
        let purpose_for_begin = purpose.clone();
        app.run_on_main_thread(move || macos::begin(token, purpose_for_begin, send))
            .map_err(|_| LocationError::new(2, "Could not start location service"))?;
        let result = tokio::time::timeout(std::time::Duration::from_secs(30), receive).await;
        let timed_out = result.is_err();
        app.run_on_main_thread(move || {
            if timed_out {
                macos::report_timeout(&request_id);
            }
            macos::cancel(&request_id);
        })
        .map_err(|_| LocationError::new(2, "Could not stop location service"))?;
        // receive yields Result<LocationFix, LocationError>; propagate it after audit.
        match result {
            Ok(Ok(fix)) => {
                match &fix {
                    Ok(_) => audit("fix", &purpose, "outcome=ok"),
                    Err(error) => audit(
                        "failure",
                        &purpose,
                        &format!("outcome=error code={}", error.code),
                    ),
                }
                fix
            }
            Ok(Err(_)) => {
                audit("failure", &purpose, "outcome=ended");
                Err(LocationError::new(2, "Location request ended"))
            }
            Err(_) => {
                audit("failure", &purpose, "outcome=timeout");
                Err(LocationError::new(3, "Location timed out"))
            }
        }
    }
    #[cfg(not(target_os = "macos"))]
    {
        let _ = (app, request_id);
        audit("rejected", &purpose, "outcome=unsupported_os");
        Err(LocationError::new(
            2,
            "Native location is available on macOS only",
        ))
    }
}

#[tauri::command]
pub fn cancel_location(app: tauri::AppHandle, request_id: String) -> Result<(), String> {
    #[cfg(target_os = "macos")]
    return app
        .run_on_main_thread(move || macos::cancel(&request_id))
        .map_err(|error| error.to_string());
    #[cfg(not(target_os = "macos"))]
    {
        let _ = (app, request_id);
        Ok(())
    }
}

#[cfg(target_os = "macos")]
mod macos {
    use super::{audit, usable_fix, LocationError, LocationFix};
    use objc2::{
        define_class, msg_send, rc::Retained, runtime::ProtocolObject, DefinedClass, MainThreadOnly,
    };
    use objc2_core_location::{
        CLAuthorizationStatus, CLError, CLLocation, CLLocationManager, CLLocationManagerDelegate,
    };
    use objc2_foundation::{MainThreadMarker, NSArray, NSError, NSObject, NSObjectProtocol};
    use std::cell::RefCell;
    use tokio::sync::oneshot;

    type Reply = oneshot::Sender<Result<LocationFix, LocationError>>;

    #[derive(Default)]
    struct DelegateState {
        reply: RefCell<Option<Reply>>,
        purpose: RefCell<String>,
    }

    define_class!(
        // NSObject has no subclassing requirements. The delegate is retained
        // with its manager until the asynchronous request has been cleaned up.
        #[unsafe(super = NSObject)]
        #[name = "OentikeLocationDelegate"]
        #[thread_kind = MainThreadOnly]
        #[ivars = DelegateState]
        struct LocationDelegate;

        unsafe impl NSObjectProtocol for LocationDelegate {}

        unsafe impl CLLocationManagerDelegate for LocationDelegate {
            #[unsafe(method(locationManager:didUpdateLocations:))]
            fn locations(&self, manager: &CLLocationManager, locations: &NSArray<CLLocation>) {
                let Some(location) = locations.lastObject() else {
                    return;
                };
                // Apple may deliver cached or invalid fixes first. Keep waiting
                // for a fresh valid measurement rather than persisting it.
                let accuracy = unsafe { location.horizontalAccuracy() };
                let timestamp = unsafe { location.timestamp() }.timeIntervalSince1970() * 1000.0;
                let now = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_secs_f64()
                    * 1000.0;
                if !usable_fix(accuracy, timestamp, now) {
                    log::debug!("[location] Ignored an invalid or expired measurement");
                    return;
                }
                let coordinate = unsafe { location.coordinate() };
                self.finish(
                    manager,
                    Ok(LocationFix {
                        latitude: coordinate.latitude,
                        longitude: coordinate.longitude,
                        accuracy,
                        timestamp,
                    }),
                );
            }

            #[unsafe(method(locationManager:didFailWithError:))]
            fn failed(&self, manager: &CLLocationManager, error: &NSError) {
                // A temporary inability to locate is recoverable; wait until
                // the request's timeout rather than treating it as a denial.
                if error.code() == CLError::LocationUnknown.0 {
                    log::info!(
                        "[location] macOS cannot currently determine a position (LocationUnknown)"
                    );
                    return;
                }
                let code = if error.code() == CLError::Denied.0 {
                    1
                } else {
                    2
                };
                self.finish(
                    manager,
                    Err(LocationError::new(
                        code,
                        &error.localizedDescription().to_string(),
                    )),
                );
            }

            #[unsafe(method(locationManagerDidChangeAuthorization:))]
            fn authorization(&self, manager: &CLLocationManager) {
                let status = unsafe { manager.authorizationStatus() };
                let purpose = self.ivars().purpose.borrow().clone();
                audit(
                    "authorization",
                    &purpose,
                    &format!("status={}", status.0),
                );
                if status == CLAuthorizationStatus::Denied
                    || status == CLAuthorizationStatus::Restricted
                {
                    self.finish(
                        manager,
                        Err(LocationError::new(1, "macOS denied location access")),
                    );
                }
            }
        }
    );

    impl LocationDelegate {
        fn new(mtm: MainThreadMarker, purpose: String, reply: Reply) -> Retained<Self> {
            let this = Self::alloc(mtm).set_ivars(DelegateState {
                reply: RefCell::new(Some(reply)),
                purpose: RefCell::new(purpose),
            });
            // NSObject's init signature is correct; allocation is on main.
            unsafe { msg_send![super(this), init] }
        }

        fn finish(&self, manager: &CLLocationManager, result: Result<LocationFix, LocationError>) {
            unsafe { manager.stopUpdatingLocation() };
            if let Some(reply) = self.ivars().reply.borrow_mut().take() {
                let _ = reply.send(result);
            }
        }
    }

    struct Request {
        id: String,
        purpose: String,
        manager: Retained<CLLocationManager>,
        // Core Location's delegate reference is weak.
        delegate: Retained<LocationDelegate>,
    }

    thread_local! { static ACTIVE: RefCell<Option<Request>> = const { RefCell::new(None) }; }

    pub fn begin(id: String, purpose: String, reply: Reply) {
        let mtm = MainThreadMarker::new().expect("location runs on Tauri's main thread");
        if ACTIVE.with(|active| active.borrow().is_some()) {
            let _ = reply.send(Err(LocationError::new(
                2,
                "Another location request is active",
            )));
            return;
        }
        // All Core Location operations stay on the main runloop.
        unsafe {
            if !CLLocationManager::locationServicesEnabled_class() {
                let _ = reply.send(Err(LocationError::new(
                    1,
                    "macOS Location Services are disabled",
                )));
                return;
            }
            let manager = CLLocationManager::new();
            let delegate = LocationDelegate::new(mtm, purpose.clone(), reply);
            manager.setDelegate(Some(ProtocolObject::from_ref(&*delegate)));
            manager.setDesiredAccuracy(10.0);
            let status = manager.authorizationStatus();
            audit(
                "start",
                &purpose,
                &format!(
                    "authorization={} (0=not determined, 1=restricted, 2=denied, 3/4=authorized)",
                    status.0
                ),
            );
            if status == CLAuthorizationStatus::Denied
                || status == CLAuthorizationStatus::Restricted
            {
                delegate.finish(
                    &manager,
                    Err(LocationError::new(1, "macOS denied location access")),
                );
                return;
            }
            ACTIVE.with(|active| {
                *active.borrow_mut() = Some(Request {
                    id,
                    purpose,
                    manager: manager.clone(),
                    delegate,
                })
            });
            if status == CLAuthorizationStatus::NotDetermined {
                manager.requestWhenInUseAuthorization();
            }
            manager.startUpdatingLocation();
        }
    }

    pub fn report_timeout(id: &str) {
        ACTIVE.with(|active| {
            let active = active.borrow();
            if let Some(request) = active.as_ref().filter(|request| request.id == id) {
                let status = unsafe { request.manager.authorizationStatus() };
                audit(
                    "timeout",
                    &request.purpose,
                    &format!("authorization={}", status.0),
                );
            }
        });
    }

    pub fn cancel(id: &str) {
        let request = ACTIVE.with(|active| {
            let mut active = active.borrow_mut();
            if active.as_ref().is_some_and(|request| request.id == id) {
                active.take()
            } else {
                None
            }
        });
        if let Some(request) = request {
            request.delegate.finish(
                &request.manager,
                Err(LocationError::new(2, "Location request cancelled")),
            );
            unsafe { request.manager.setDelegate(None) };
        }
    }
}

#[cfg(test)]
mod location_tests {
    use super::{path_is_app_bundle, usable_fix};
    use std::path::Path;

    #[test]
    fn recent_cached_position_is_accepted_without_rewriting_timestamp() {
        assert!(usable_fix(12.0, 95_000.0, 100_000.0));
        assert!(usable_fix(12.0, 70_000.0, 100_000.0));
    }

    #[test]
    fn expired_and_invalid_positions_remain_rejected() {
        assert!(!usable_fix(12.0, 69_999.0, 100_000.0));
        assert!(!usable_fix(-1.0, 100_000.0, 100_000.0));
        assert!(!usable_fix(f64::NAN, 100_000.0, 100_000.0));
        assert!(!usable_fix(12.0, f64::NAN, 100_000.0));
        assert!(!usable_fix(12.0, 0.0, 100_000.0));
    }

    #[test]
    fn only_macos_app_bundle_paths_are_trusted_hosts() {
        assert!(path_is_app_bundle(Path::new(
            "/tmp/Oentike.app/Contents/MacOS/Oentike"
        )));
        assert!(!path_is_app_bundle(Path::new(
            "/tmp/target/debug/Oentike"
        )));
        assert!(!path_is_app_bundle(Path::new(
            "/tmp/Oentike.app/Contents/Resources/helper"
        )));
    }
}
