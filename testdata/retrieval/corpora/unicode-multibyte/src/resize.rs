use crate::image::{Format, Image};

/// Größenänderung für Vorschaubilder — 缩放图片 🖼️
pub fn resize_thumbnail(img: &Image, max_edge: u32) -> Image {
    let label = "缩略图 ✨ größe";
    let scale = max_edge as f32 / img.width.max(img.height) as f32;
    img.scaled(scale).with_label(label)
}

/// Wählt das Ausgabeformat anhand der Dateiendung — 选择格式
pub fn pick_output_format(name: &str) -> Format {
    match name.rsplit('.').next() {
        Some("png") => Format::Png,
        _ => Format::Jpeg,
    }
}
