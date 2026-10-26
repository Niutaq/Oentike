fn main() {
    println!("cargo:rerun-if-env-changed=OENTIKE_PACK_PUBLIC_KEY_HEX");
    println!("cargo:rerun-if-changed=../../oentike-proto/conditions.proto");

    println!("cargo:rerun-if-changed=icons/icon.icns");
    println!("cargo:rerun-if-changed=icons/icon.ico");
    println!("cargo:rerun-if-changed=icons/icon.png");
    println!("cargo:rerun-if-changed=icons/32x32.png");
    println!("cargo:rerun-if-changed=icons/128x128.png");
    println!("cargo:rerun-if-changed=icons/128x128@2x.png");
    println!("cargo:rerun-if-changed=tauri.conf.json");
    println!("cargo:rerun-if-changed=Info.plist");
    println!("cargo:rerun-if-changed=capabilities/default.json");

    // Generate a gRPC client for the product conditions service.
    tonic_build::configure()
        .build_client(true)
        .build_server(false)
        .compile_protos(
            &["../../oentike-proto/conditions.proto"],
            &["../../oentike-proto"],
        )
        .expect("compile conditions.proto");

    // Remote origins (Astro on 127.0.0.1:4321) are ACL-checked. Without an
    // AppManifest + allow-* permissions, get_conditions_ui and friends are denied
    // and the UI stays empty while /api/readyz still succeeds via fetch.
    tauri_build::try_build(
        tauri_build::Attributes::new().app_manifest(
            tauri_build::AppManifest::new().commands(&[
                "get_conditions_ui",
                "get_season_ui",
                "search_cells_ui",
                "ensure_cell_ui",
                "import_offline_pack",
                "get_offline_pack",
                "open_location_settings",
                "location_host_status",
                "get_current_location",
                "cancel_location",
            ]),
        ),
    )
    .expect("tauri build");
}
