tonic::include_proto!("oentike.conditions");

use serde::Serialize;
use tonic::transport::Endpoint;

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct FactorOut {
    id: String,
    unit: String,
    value: Option<f64>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct ConditionsOut {
    cell_id: String,
    cell_name: String,
    species_slug: String,
    target_date: String,
    status: String,
    score: Option<i32>,
    confidence: String,
    factors: Vec<FactorOut>,
    algorithm_version: String,
    input_sha256: Option<String>,
    fetched_at: Option<String>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct SeasonDayOut {
    date: String,
    status: String,
    score: Option<i32>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct SeasonOut {
    cell_id: String,
    cell_name: String,
    species_slug: String,
    algorithm_version: String,
    days: Vec<SeasonDayOut>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct CellSummaryOut {
    id: String,
    name: String,
    lon: f64,
    lat: f64,
    west: f64,
    south: f64,
    east: f64,
    north: f64,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct SearchCellsOut {
    cells: Vec<CellSummaryOut>,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct EnsureCellOut {
    cell: CellSummaryOut,
    ingested: bool,
}

fn cell_summary_out(c: CellSummary) -> CellSummaryOut {
    CellSummaryOut {
        id: c.id,
        name: c.name,
        lon: c.lon,
        lat: c.lat,
        west: c.west,
        south: c.south,
        east: c.east,
        north: c.north,
    }
}

mod commands {
    use super::*;

    #[tauri::command]
    pub async fn get_conditions_ui(
        cell_id: String,
        species_slug: Option<String>,
        target_date: Option<String>,
        grpc_addr: Option<String>,
    ) -> Result<ConditionsOut, String> {
        let grpc_addr =
            grpc_addr.unwrap_or_else(|| "http://127.0.0.1:8082".to_string());
        let endpoint = Endpoint::from_shared(grpc_addr)
            .map_err(|e| format!("parse grpc endpoint: {e}"))?;

        let mut client = conditions_service_client::ConditionsServiceClient::connect(endpoint)
            .await
            .map_err(|e| format!("connect grpc: {e}"))?;

        let req = GetConditionsRequest {
            cell_id,
            species_slug: species_slug.unwrap_or_default(),
            target_date: target_date.unwrap_or_default(),
        };

        let resp = client
            .get_conditions(req)
            .await
            .map_err(|e| format!("grpc get_conditions: {e}"))?
            .into_inner();

        let factors = resp
            .factors
            .into_iter()
            .map(|f| FactorOut {
                id: f.id,
                unit: f.unit,
                value: f.value,
            })
            .collect();

        Ok(ConditionsOut {
            cell_id: resp.cell_id,
            cell_name: resp.cell_name,
            species_slug: resp.species_slug,
            target_date: resp.target_date,
            status: resp.status,
            score: resp.score,
            confidence: resp.confidence,
            factors,
            algorithm_version: resp.algorithm_version,
            input_sha256: resp.input_sha256,
            fetched_at: resp.fetched_at,
        })
    }

    #[tauri::command]
    pub async fn get_season_ui(
        cell_id: String,
        species_slug: Option<String>,
        days: Option<i32>,
        grpc_addr: Option<String>,
    ) -> Result<SeasonOut, String> {
        let grpc_addr =
            grpc_addr.unwrap_or_else(|| "http://127.0.0.1:8082".to_string());
        let endpoint = Endpoint::from_shared(grpc_addr)
            .map_err(|e| format!("parse grpc endpoint: {e}"))?;

        let mut client = conditions_service_client::ConditionsServiceClient::connect(endpoint)
            .await
            .map_err(|e| format!("connect grpc: {e}"))?;

        let req = GetSeasonRequest {
            cell_id,
            species_slug: species_slug.unwrap_or_default(),
            days: days.unwrap_or(0),
        };

        let resp = client
            .get_season(req)
            .await
            .map_err(|e| format!("grpc get_season: {e}"))?
            .into_inner();

        let days = resp
            .days
            .into_iter()
            .map(|d| SeasonDayOut {
                date: d.date,
                status: d.status,
                score: d.score,
            })
            .collect();

        Ok(SeasonOut {
            cell_id: resp.cell_id,
            cell_name: resp.cell_name,
            species_slug: resp.species_slug,
            algorithm_version: resp.algorithm_version,
            days,
        })
    }

    #[tauri::command]
    pub async fn search_cells_ui(
        query: Option<String>,
        limit: Option<i32>,
        grpc_addr: Option<String>,
    ) -> Result<SearchCellsOut, String> {
        let grpc_addr =
            grpc_addr.unwrap_or_else(|| "http://127.0.0.1:8082".to_string());
        let endpoint = Endpoint::from_shared(grpc_addr)
            .map_err(|e| format!("parse grpc endpoint: {e}"))?;

        let mut client = conditions_service_client::ConditionsServiceClient::connect(endpoint)
            .await
            .map_err(|e| format!("connect grpc: {e}"))?;

        let req = SearchCellsRequest {
            query: query.unwrap_or_default(),
            limit: limit.unwrap_or(0),
        };

        let resp = client
            .search_cells(req)
            .await
            .map_err(|e| format!("grpc search_cells: {e}"))?
            .into_inner();

        let cells = resp
            .cells
            .into_iter()
            .map(cell_summary_out)
            .collect();

        Ok(SearchCellsOut { cells })
    }

    #[tauri::command]
    pub async fn ensure_cell_ui(
        cell_id: String,
        grpc_addr: Option<String>,
    ) -> Result<EnsureCellOut, String> {
        let grpc_addr =
            grpc_addr.unwrap_or_else(|| "http://127.0.0.1:8082".to_string());
        let endpoint = Endpoint::from_shared(grpc_addr)
            .map_err(|e| format!("parse grpc endpoint: {e}"))?;

        let mut client = conditions_service_client::ConditionsServiceClient::connect(endpoint)
            .await
            .map_err(|e| format!("connect grpc: {e}"))?;

        let resp = client
            .ensure_cell(EnsureCellRequest { cell_id })
            .await
            .map_err(|e| format!("grpc ensure_cell: {e}"))?
            .into_inner();

        let cell = resp
            .cell
            .ok_or_else(|| "grpc ensure_cell: missing cell".to_string())?;

        Ok(EnsureCellOut {
            cell: cell_summary_out(cell),
            ingested: resp.ingested,
        })
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            commands::get_conditions_ui,
            commands::get_season_ui,
            commands::search_cells_ui,
            commands::ensure_cell_ui
        ])
        .setup(|app| {
            if cfg!(debug_assertions) {
                app.handle().plugin(
                    tauri_plugin_log::Builder::default()
                        .level(log::LevelFilter::Info)
                        .build(),
                )?;
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
