use proxy_wasm::traits::*;
use proxy_wasm::types::*;
use log::{info, warn};

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {
        Box::new(FinOpsRootContext)
    });
}}

struct FinOpsRootContext;

impl Context for FinOpsRootContext {}
impl RootContext for FinOpsRootContext {
    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }

    fn create_http_context(&self, context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(FinOpsHttpContext {
            context_id,
        }))
    }
}

struct FinOpsHttpContext {
    context_id: u32,
}

impl Context for FinOpsHttpContext {}

impl HttpContext for FinOpsHttpContext {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        let path = self.get_http_request_header(":path").unwrap_or_else(|| String::from("unknown"));
        info!("FinOps Tollbooth: Zarejestrowano żądanie wejściowe! (context {}: path: {})", self.context_id, path);

        let budget_header = self.get_http_request_header("x-finops-budget");
        let cost_header = self.get_http_request_header("x-finops-cost");

        if let (Some(budget_str), Some(cost_str)) = (budget_header, cost_header) {
            let budget: f64 = budget_str.parse().unwrap_or(0.0);
            let cost: f64 = cost_str.parse().unwrap_or(0.0);

            info!("FinOps Tollbooth: Analiza budżetu - Koszt: {}, Dostępny Budżet: {}", cost, budget);

            if cost > budget {
                warn!("FinOps Tollbooth: PRZEKROCZONO BUDŻET! Odrzucanie żądania (HTTP 402)");
                self.send_http_response(
                    402,
                    vec![("Powered-By", "Oentike-WASM-FinOps")],
                    Some(b"Zbyt maly budzet na te operacje (FinOps Tollbooth)"),
                );
                return Action::Pause;
            } else {
                info!("FinOps Tollbooth: Budżet zweryfikowany pozytywnie. Przepuszczam.");
            }
        } else {
            info!("FinOps Tollbooth: Brak pełnych nagłówków FinOps. Przepuszczam (lub można tu wymusić Zero-Trust i zablokować).");
        }

        Action::Continue
    }
    
    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        Action::Continue
    }
}
