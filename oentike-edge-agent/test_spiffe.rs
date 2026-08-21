use spiffe::workload_api::client::WorkloadApiClient;

#[tokio::main]
async fn main() {
    let client = WorkloadApiClient::default().await.unwrap();
    let ctx = client.fetch_x509_context().await.unwrap();
    let svid = ctx.default_svid().unwrap();
    println!("{}", svid.spiffe_id());
}
