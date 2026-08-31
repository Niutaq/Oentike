use std::sync::{
    atomic::{AtomicBool, Ordering},
    Arc,
};
use sysinfo::System;
use tauri::Emitter;
use tokio::sync::Mutex;

pub mod fingate {
    tonic::include_proto!("oentike.fingate");
}

pub struct AppState {
    pub sys: Mutex<System>,
    pub live_connected: AtomicBool,
}

#[derive(Clone, serde::Serialize)]
struct SystemStats {
    cpu: f32,
    used_ram_gb: f32,
    total_ram_gb: f32,
}

#[tauri::command]
async fn get_system_stats(state: tauri::State<'_, Arc<AppState>>) -> Result<SystemStats, String> {
    let mut sys = state.sys.lock().await;
    sys.refresh_cpu_usage();
    sys.refresh_memory();
    let cpu = sys.global_cpu_info().cpu_usage();
    let bytes_per_gib = 1024.0 * 1024.0 * 1024.0;
    let used_ram_gb = sys.used_memory() as f32 / bytes_per_gib;
    let total_ram_gb = sys.total_memory() as f32 / bytes_per_gib;
    Ok(SystemStats {
        cpu,
        used_ram_gb,
        total_ram_gb,
    })
}

#[tauri::command]
fn get_live_status(state: tauri::State<'_, Arc<AppState>>) -> bool {
    state.live_connected.load(Ordering::Acquire)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let mut initial_sys = System::new_all();
    initial_sys.refresh_cpu_usage(); // initial poll

    let state = Arc::new(AppState {
        sys: Mutex::new(initial_sys),
        live_connected: AtomicBool::new(false),
    });

    tauri::Builder::default()
        .manage(state.clone())
        .setup(move |app| {
            if cfg!(debug_assertions) {
                app.handle().plugin(
                    tauri_plugin_log::Builder::default()
                        .level(log::LevelFilter::Info)
                        .build(),
                )?;
            }

            let app_handle = app.handle().clone();
            let live_state = state.clone();
            tauri::async_runtime::spawn(async move {
                use futures::stream::StreamExt;

                loop {
                    let subscriptions = async {
                        let client = async_nats::connect("nats://127.0.0.1:4222").await?;
                        let ai = client.subscribe("SECOPS.ai_processed").await?;
                        let metrics = client.subscribe("SECOPS.metrics").await?;
                        Ok::<_, async_nats::Error>((ai, metrics))
                    }
                    .await;

                    if let Ok((mut ai, mut metrics)) = subscriptions {
                        live_state.live_connected.store(true, Ordering::Release);
                        let _ = app_handle.emit("live-status", true);

                        loop {
                            tokio::select! {
                                message = ai.next() => {
                                    let Some(message) = message else { break };
                                    if let Ok(payload) = String::from_utf8(message.payload.to_vec()) {
                                        let _ = app_handle.emit("review-event", payload);
                                    }
                                }
                                message = metrics.next() => {
                                    let Some(message) = message else { break };
                                    if let Ok(payload) = String::from_utf8(message.payload.to_vec()) {
                                        let _ = app_handle.emit("activity-event", payload);
                                    }
                                }
                            }
                        }
                    }

                    live_state.live_connected.store(false, Ordering::Release);
                    let _ = app_handle.emit("live-status", false);
                    tokio::time::sleep(std::time::Duration::from_secs(2)).await;
                }
            });

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![get_system_stats, get_live_status])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
