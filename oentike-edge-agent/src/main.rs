// Using elements:
use iced::{
    widget::{button, column, container, text},
    Alignment, Application, Command, Element, Length, Settings, Theme,
};

// Re-exports gRPC types from the generated protobuf module.
pub mod fingate {
    tonic::include_proto!("oentike.fingate");
}
use fingate::fingate_service_client::FingateServiceClient;
use fingate::{UsageRequest, UsageResponse};

// Entry point of the application.
pub fn main() -> iced::Result {
    FinGateAgent::run(Settings {
        window: iced::window::Settings {
            size: iced::Size::new(600.0, 400.0),
            ..Default::default()
        },
        ..Default::default()
    })
}

// A struct representing the main application state.
struct FinGateAgent {
    status_text: String,
    is_loading: bool,
}

// Enum representing the possible messages that can be sent to the application.
#[derive(Debug, Clone)]
enum Message {
    SendRequest,
    ResponseReceived(Result<UsageResponse, String>),
}

// The main application struct.
impl Application for FinGateAgent {
    type Executor = iced::executor::Default;
    type Message = Message;
    type Theme = Theme;
    type Flags = ();

    // Initializes the application state and returns the initial view and command.
    fn new(_flags: ()) -> (Self, Command<Message>) {
        (
            FinGateAgent {
                status_text: "Ready to send gRPC request\n(Standard FinOps FOCUS)".into(),
                is_loading: false,
            },
            Command::none(),
        )
    }

    // Returns the title of the application.
    fn title(&self) -> String {
        String::from("Oentike")
    }

    // Updates the application state based on incoming messages.
    fn update(&mut self, message: Message) -> Command<Message> {
        match message {
            Message::SendRequest => {
                self.is_loading = true;
                self.status_text = "Connecting to Control Plane...".into();
                Command::perform(send_grpc_request(), Message::ResponseReceived)
            }
            Message::ResponseReceived(Ok(response)) => {
                self.is_loading = false;
                self.status_text = format!(
                    "Got: CPU: {} Memory: {}",
                    response.cpu, response.memory
                );
                Command::none()
            }
            Message::ResponseReceived(Err(e)) => {
                self.is_loading = false;
                self.status_text = format!("Error: {}", e);
                Command::none()
            }
        }
    }

    fn view(&self) -> Element<'_, Message> {
        let title = text("Oentike")
            .size(32);

        let subtitle = text("Zero-Trust Edge Node")
            .size(16)
            .style(iced::theme::Text::Color(iced::Color::from_rgb(0.5, 0.5, 0.5)));

        let mut send_btn = button(text("Send request to server").size(18))
            .padding(15)
            .style(iced::theme::Button::Primary);

        if !self.is_loading {
            send_btn = send_btn.on_press(Message::SendRequest);
        }

        let status_display = text(&self.status_text)
            .size(20)
            .horizontal_alignment(iced::alignment::Horizontal::Center);

        let content = column![title, subtitle, send_btn, status_display]
            .spacing(30)
            .align_items(Alignment::Center);

        container(content)
            .width(Length::Fill)
            .height(Length::Fill)
            .center_x()
            .center_y()
            .padding(20)
            .into()
    }

    fn theme(&self) -> Theme {
        Theme::Dark
    }
}

// Sends a gRPC request to the server and returns the response
async fn send_grpc_request() -> Result<UsageResponse, String> {
    let mut client = FingateServiceClient::connect("http://127.0.0.1:50051")
        .await
        .map_err(|e| format!("gRPC Connect Error: {}", e))?;

    let request = tonic::Request::new(UsageRequest {
        agent_id: "edge-agent-ui-001".into(),
    });

    let response = client
        .get_usage(request)
        .await
        .map_err(|e| format!("gRPC Request Error: {}", e))?;

    Ok(response.into_inner())
}
