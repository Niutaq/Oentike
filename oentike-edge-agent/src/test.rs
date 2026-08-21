use iced::widget::text_editor;
use iced::Length;

fn test() {
    let mut content = text_editor::Content::new();
    let te = text_editor(&content).height(Length::Fixed(100.0));
}
