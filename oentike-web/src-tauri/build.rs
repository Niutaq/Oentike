fn main() {
    println!("cargo:rerun-if-changed=../../oentike-proto/conditions.proto");

    println!("cargo:rerun-if-changed=icons/icon.icns");
    println!("cargo:rerun-if-changed=icons/icon.ico");
    println!("cargo:rerun-if-changed=icons/icon.png");
    println!("cargo:rerun-if-changed=icons/32x32.png");
    println!("cargo:rerun-if-changed=icons/128x128.png");
    println!("cargo:rerun-if-changed=icons/128x128@2x.png");
    println!("cargo:rerun-if-changed=tauri.conf.json");

    // Generate a gRPC client for the product conditions service.
    tonic_build::configure()
        .build_client(true)
        .build_server(false)
        .compile_protos(
            &["../../oentike-proto/conditions.proto"],
            &["../../oentike-proto"],
        )
        .expect("compile conditions.proto");

    tauri_build::build();
}
