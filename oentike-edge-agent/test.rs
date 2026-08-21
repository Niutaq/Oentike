use iced::{
    widget::text_editor,
    Length,
};
pub fn test_ui() {
    let content = text_editor::Content::new();
    let te = text_editor(&content).height(Length::Fixed(100.0));
}
