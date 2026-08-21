use iced::{
    widget::{button, column, container, row, text, scrollable, Space, Rule},
    Alignment, Application, Command, Element, Length, Settings, Theme, Color, Font
};
use sysinfo::System;

pub mod fingate {
    tonic::include_proto!("oentike.fingate");
}
use fingate::fingate_service_client::FingateServiceClient;
use fingate::{TelemetryRequest, TelemetryResponse};

#[derive(Debug, Clone, Copy, PartialEq)]
enum Language {
    PL,
    EN,
}

#[derive(Debug, Clone, Copy, PartialEq)]
enum ThemeMode {
    Light,
    Dark,
}

fn i18n(key: &str, lang: Language) -> String {
    match lang {
        Language::PL => match key {
            "ui.form.header" => "Zero-Trust Hardware Sensor",
            "ui.form.button.submit" => "Start Telemetry Stream",
            "ui.form.status.idle" => "Oczekuje",
            "ui.window.title" => "Oentike (FinOps Proxy)",
            "ui.form.status.connecting" => "Strumieniowanie telemetrii...",
            "ui.form.status.decision" => "Status",
            "ui.form.status.error" => "Błąd",
            "ui.form.subtitle" => "Transmisja telemetrii sprzętowej w czasie rzeczywistym via mTLS Zero-Trust do Control Plane.",
            "PENDING_AI" => "OCZEKUJE NA AI",
            "Wniosek został przekazany do analizy LLM przez kolejkę NATS JetStream." => "Zadanie przekazane do analizy FinOps.",
            "ui.toggle.lang" => "PL",
            "ui.toggle.theme" => "Zmień motyw",
            "ui.form.button.web" => "Otwórz witrynę wizualizacji",
            _ => key,
        },
        Language::EN => match key {
            "ui.form.header" => "Zero-Trust Hardware Sensor",
            "ui.form.button.submit" => "Start Telemetry Stream",
            "ui.form.status.idle" => "Waiting to start",
            "ui.window.title" => "Oentike (FinOps Proxy)",
            "ui.form.status.connecting" => "Streaming telemetry...",
            "ui.form.status.decision" => "Status",
            "ui.form.status.error" => "Error",
            "ui.form.subtitle" => "Real-time hardware telemetry streaming via mTLS Zero-Trust to Control Plane.",
            "PENDING_AI" => "PENDING AI",
            "Wniosek został przekazany do analizy LLM przez kolejkę NATS JetStream." => "Event routed to AI pipeline.",
            "ui.toggle.lang" => "EN",
            "ui.toggle.theme" => "Toggle Theme",
            "ui.form.button.web" => "Open Visualizations Web",
            _ => key,
        },
    }.to_string()
}

pub fn main() -> iced::Result {
    FinGateAgent::run(Settings {
        window: iced::window::Settings {
            size: iced::Size::new(550.0, 750.0),
            min_size: Some(iced::Size::new(450.0, 600.0)),
            ..Default::default()
        },
        antialiasing: false,
        ..Default::default()
    })
}

struct FinGateAgent {
    raw_status_key: String,
    is_loading: bool,
    is_error: bool,

    language: Language,
    theme_mode: ThemeMode,
}

#[derive(Debug, Clone)]
enum Message {
    SendRequest,
    ResponseReceived(Result<TelemetryResponse, String>),
    ToggleLanguage,
    ToggleTheme,
    OpenWebVisualizer,
}

impl Application for FinGateAgent {
    type Executor = iced::executor::Default;
    type Message = Message;
    type Theme = Theme;
    type Flags = ();

    fn new(_flags: ()) -> (Self, Command<Message>) {
        (
            FinGateAgent {
                raw_status_key: "ui.form.status.idle".into(),
                is_loading: false,
                is_error: false,
                language: Language::EN,
                theme_mode: ThemeMode::Dark,
            },
            Command::none(),
        )
    }

    fn title(&self) -> String {
        i18n("ui.window.title", self.language)
    }

    fn update(&mut self, message: Message) -> Command<Message> {
        match message {
            Message::ToggleLanguage => {
                self.language = if self.language == Language::PL { Language::EN } else { Language::PL };
                Command::none()
            }

            Message::ToggleTheme => {
                self.theme_mode = if self.theme_mode == ThemeMode::Dark { ThemeMode::Light } else { ThemeMode::Dark };
                Command::none()
            }

            Message::OpenWebVisualizer => {
                std::process::Command::new("xdg-open")
                    .arg("http://localhost:4321/")
                    .spawn()
                    .ok();
                Command::none()
            }

            Message::SendRequest => {
                println!("[UI] Button 'Start Telemetry Stream' clicked. Initializing connection...");
                self.is_loading = true;
                self.is_error = false;
                self.raw_status_key = "ui.form.status.connecting".into();

                Command::perform(start_telemetry_stream(), Message::ResponseReceived)
            }
            Message::ResponseReceived(Ok(response)) => {
                println!("[UI] Success! Response status: {}", response.status);
                self.is_loading = false;
                self.is_error = false;
                self.raw_status_key = response.status;
                Command::none()
            }
            Message::ResponseReceived(Err(e)) => {
                println!("[UI] Error encountered: {}", e);
                self.is_loading = false;
                self.is_error = true;
                self.raw_status_key = e;
                Command::none()
            }
        }
    }

    fn view(&self) -> Element<'_, Message> {
        let default_font = Font::with_name("Helvetica Neue");

        let lang_btn = button(text(i18n("ui.toggle.lang", self.language)).font(default_font).size(14))
            .on_press(Message::ToggleLanguage)
            .padding([8, 16])
            .style(iced::theme::Button::Secondary);

        let theme_btn = button(text(i18n("ui.toggle.theme", self.language)).font(default_font).size(14))
            .on_press(Message::ToggleTheme)
            .padding([8, 16])
            .style(iced::theme::Button::Secondary);

        let top_bar = row![Space::with_width(Length::Fill), theme_btn, lang_btn]
            .spacing(10)
            .width(Length::Fill)
            .align_items(Alignment::Center);

        let title = text(i18n("ui.form.header", self.language))
            .size(32)
            .font(default_font);

        let subtitle = text(i18n("ui.form.subtitle", self.language))
            .size(14)
            .style(iced::theme::Text::Color(Color::from_rgb(0.5, 0.5, 0.5)))
            .font(default_font);

        let header = column![title, subtitle].spacing(8);

        let mut send_btn = button(
                text(i18n("ui.form.button.submit", self.language))
                    .size(16)
                    .font(default_font)
                    .horizontal_alignment(iced::alignment::Horizontal::Center)
            )
            .padding([14, 24])
            .width(Length::Fill)
            .style(iced::theme::Button::Primary);

        if !self.is_loading {
            send_btn = send_btn.on_press(Message::SendRequest);
        }

        let display_status = if self.is_loading || self.raw_status_key == "ui.form.status.idle" || self.raw_status_key == "ui.form.status.error" {
            i18n(&self.raw_status_key, self.language)
        } else {
            i18n(&self.raw_status_key, self.language)
        };

        let status_color = if self.is_error {
            Color::from_rgb(0.9, 0.3, 0.3)
        } else if self.raw_status_key == "ui.form.status.idle" {
            Color::from_rgb(0.5, 0.5, 0.5)
        } else {
            Color::from_rgb(0.2, 0.8, 0.4)
        };

        let status_display = row![
            text(format!("{}:", i18n("ui.form.status.decision", self.language))).size(14).font(default_font),
            text(display_status).size(14).font(default_font).style(iced::theme::Text::Color(status_color))
        ].spacing(8).align_items(Alignment::Center);

        let status_card = column![status_display].spacing(10);

        let web_btn = button(
                text(i18n("ui.form.button.web", self.language))
                    .size(16)
                    .font(default_font)
                    .horizontal_alignment(iced::alignment::Horizontal::Center)
            )
            .padding([14, 24])
            .width(Length::Fill)
            .style(iced::theme::Button::Secondary)
            .on_press(Message::OpenWebVisualizer);

        let form_content = column![
            header,
            Space::with_height(30),
            send_btn,
            Space::with_height(10),
            web_btn,
            Space::with_height(20),
            Rule::horizontal(1),
            Space::with_height(10),
            status_card
        ]
        .spacing(15)
        .max_width(450);

        let main_col = column![top_bar, form_content]
            .spacing(30)
            .width(Length::Fill)
            .align_items(Alignment::Center);

        let scroll = scrollable(
            container(main_col)
                .width(Length::Fill)
                .center_x()
                .padding(30)
        )
        .width(Length::Fill)
        .height(Length::Fill);

        container(scroll)
            .width(Length::Fill)
            .height(Length::Fill)
            .center_y()
            .into()
    }

    fn theme(&self) -> Theme {
        match self.theme_mode {
            ThemeMode::Light => Theme::Light,
            ThemeMode::Dark => Theme::Dark,
        }
    }
}

async fn start_telemetry_stream() -> Result<TelemetryResponse, String> {
    println!("[Agent] Attempting to connect to Workload API socket at /tmp/spire-sockets/workload_api.sock...");
    let start_time = std::time::Instant::now();

    let mut workload_client = None;
    for _ in 0..30 {
        match spiffe::workload_api::client::WorkloadApiClient::connect_to("unix:///tmp/spire-sockets/workload_api.sock").await {
            Ok(client) => {
                workload_client = Some(client);
                break;
            }
            Err(e) => {
                println!("[Agent] Waiting for Workload API socket... ({})", e);
                tokio::time::sleep(std::time::Duration::from_secs(1)).await;
            }
        }
    }
    let workload_client = workload_client.ok_or_else(|| {
        println!("[Agent] Failed to connect to Workload API within 30 seconds.");
        "SPIFFE Error: Failed to connect to Workload API after 30s.".to_string()
    })?;
    println!("[Agent] Successfully connected to Workload API.");

    println!("[Agent] Fetching X.509 context...");
    let ctx = workload_client.fetch_x509_context().await
        .map_err(|e| {
            println!("[Agent] Error fetching X.509 context: {}", e);
            format!("SPIFFE Error: Failed to fetch X509 context: {}", e)
        })?;
    println!("[Agent] Successfully fetched X.509 context.");

    let svid = ctx.default_svid()
        .ok_or_else(|| "SPIFFE Error: No default SVID found".to_string())?;

    let mut certs_pem = String::new();
    for cert in svid.cert_chain() {
        certs_pem.push_str(&pem::encode(&pem::Pem::new("CERTIFICATE", cert.as_bytes())));
    }
    let key_pem = pem::encode(&pem::Pem::new("PRIVATE KEY", svid.private_key().as_bytes()));
    let identity = tonic::transport::Identity::from_pem(certs_pem, key_pem);

    let bundles = ctx.bundle_set();
    let bundle = bundles.get(svid.spiffe_id().trust_domain())
        .ok_or_else(|| "SPIFFE Error: No trust bundle found for trust domain".to_string())?;

    let mut ca_pem = String::new();
    for auth in bundle.authorities() {
        ca_pem.push_str(&pem::encode(&pem::Pem::new("CERTIFICATE", auth.as_bytes())));
    }
    let ca_certificate = tonic::transport::Certificate::from_pem(ca_pem);

    let tls_config = tonic::transport::ClientTlsConfig::new()
        .identity(identity)
        .ca_certificate(ca_certificate)
        .domain_name("localhost");

    println!("[Agent] Connecting to Envoy mTLS proxy at https://localhost:10000...");
    let mut channel = None;
    for _ in 0..30 {
        match tonic::transport::Channel::from_static("https://localhost:10000")
            .tls_config(tls_config.clone())
            .map_err(|e| format!("TLS Config Error: {}", e))?
            .connect()
            .await 
        {
            Ok(c) => {
                channel = Some(c);
                break;
            }
            Err(e) => {
                println!("[Agent] Waiting for Envoy proxy... ({})", e);
                tokio::time::sleep(std::time::Duration::from_secs(1)).await;
            }
        }
    }
    let channel = channel.ok_or_else(|| {
        println!("[Agent] Failed to connect to Envoy proxy within 30 seconds.");
        "Connection Failed. Envoy proxy not reachable after 30s.".to_string()
    })?;
    println!("[Agent] Connected to Envoy. Starting telemetry stream...");

    let mtls_setup_ms = start_time.elapsed().as_millis() as i64;
    let agent_id = svid.spiffe_id().to_string();

    let mut client = FingateServiceClient::new(channel);

    let (tx, rx) = tokio::sync::mpsc::channel(10);

    tokio::spawn(async move {
        let mut sys = System::new_all();
        loop {
            sys.refresh_cpu_usage();
            sys.refresh_memory();
            tokio::time::sleep(std::time::Duration::from_millis(500)).await;
            sys.refresh_cpu_usage();
            
            let cpu_usage_percent = sys.global_cpu_info().cpu_usage();
            let memory_used_mb = (sys.used_memory() / 1024 / 1024) as i32;

            let req = TelemetryRequest {
                agent_id: agent_id.clone(),
                cpu_usage_percent,
                memory_used_mb,
                metrics: Some(fingate::ClientMetrics {
                    mtls_setup_ms,
                    payload_bytes: 128,
                    region: "eu-central-1".to_string(),
                }),
            };
            
            if tx.send(req).await.is_err() {
                break;
            }
            tokio::time::sleep(std::time::Duration::from_secs(2)).await;
        }
    });

    let stream = tokio_stream::wrappers::ReceiverStream::new(rx);
    let mut request = tonic::Request::new(stream);
    request.metadata_mut().insert("x-finops-cost", tonic::metadata::MetadataValue::from_static("50.0"));
    request.metadata_mut().insert("x-finops-budget", tonic::metadata::MetadataValue::from_static("100.0"));

    let response = client
        .stream_telemetry(request)
        .await
        .map_err(|e| format!("Request Failed: {}", e))?;

    Ok(response.into_inner())
}
