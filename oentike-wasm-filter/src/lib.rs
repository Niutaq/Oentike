use log::{error, info, warn};
use proxy_wasm::traits::*;
use proxy_wasm::types::*;
use regex::Regex;

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {
        Box::new(AiGatewayRootContext)
    });
}}

struct AiGatewayRootContext;

impl Context for AiGatewayRootContext {}
impl RootContext for AiGatewayRootContext {
    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }

    fn create_http_context(&self, context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(AiGatewayHttpContext {
            context_id,
            body_buffer: Vec::new(),
        }))
    }
}

struct AiGatewayHttpContext {
    context_id: u32,
    body_buffer: Vec<u8>,
}

impl Context for AiGatewayHttpContext {}

impl HttpContext for AiGatewayHttpContext {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        info!(
            "Oentike AI-Mesh: Intercepting request (Context: {})",
            self.context_id
        );

        // CRITICAL: We remove Content-Length because our PII redaction might change the payload size.
        // Envoy will automatically recalculate it or fall back to Transfer-Encoding: chunked.
        self.set_http_request_header("content-length", None);

        Action::Continue
    }

    fn on_http_request_body(&mut self, body_size: usize, end_of_stream: bool) -> Action {
        // 1. Buffer the incoming HTTP chunks
        if let Some(mut chunk) = self.get_http_request_body(0, body_size) {
            self.body_buffer.append(&mut chunk);
        }

        // 2. If the stream isn't finished, pause and tell Envoy to give us the next chunk
        if !end_of_stream {
            return Action::Pause;
        }

        // 3. We have the full prompt. Let's analyze it.
        let body_str = match String::from_utf8(self.body_buffer.clone()) {
            Ok(s) => s,
            Err(_) => {
                error!("Oentike AI-Mesh: Non-UTF8 payload detected, bypassing redaction.");
                return Action::Continue;
            }
        };

        // 4. Fast-Path Regex (Compiled directly into WASM for 0-latency)
        // Matches typical AWS Access Keys (AKIA...)
        let re = Regex::new(r"AKIA[0-9A-Z]{16}").unwrap();
        let redacted = re.replace_all(&body_str, "[REDACTED_AWS_KEY]");

        if redacted != body_str {
            warn!("Oentike AI-Mesh: PII (AWS Key) detected and redacted in prompt!");

            let telemetry_payload =
                r#"{"type":"AWS_KEY_BLOCKED", "event":"PII_REDACTED", "agent_id":"wasm-filter"}"#;

            let _ = self
                .dispatch_http_call(
                    "telemetry_service",
                    vec![
                        (":method", "POST"),
                        (":path", "/telemetry"),
                        (":authority", "telemetry_service"),
                        ("content-type", "application/json"),
                    ],
                    Some(telemetry_payload.as_bytes()),
                    vec![],
                    std::time::Duration::from_millis(500),
                )
                .map_err(|e| error!("Failed to dispatch HTTP callout: {:?}", e));
        }

        // 5. Inject the scrubbed payload back into the Envoy request pipeline
        let new_body_bytes = redacted.as_bytes();
        self.set_http_request_body(0, self.body_buffer.len(), new_body_bytes);

        Action::Continue
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        Action::Continue
    }
}
