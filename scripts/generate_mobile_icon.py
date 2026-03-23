from pathlib import Path
from PIL import Image, ImageDraw


ROOT = Path(__file__).resolve().parents[1]
SRC_ICON = ROOT / "client" / "app-icon.png"
OUT_DIR = ROOT / "mobile" / "resources"
OUT_ICON = OUT_DIR / "icon.png"
OUT_SPLASH = OUT_DIR / "splash.png"


def create_icon_canvas(size: int = 1024) -> Image.Image:
    canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(canvas)

    margin = int(size * 0.07)
    rect = [margin, margin, size - margin, size - margin]
    radius = int(size * 0.20)

    # Ancient-scroll-like warm white rounded background
    draw.rounded_rectangle(
        rect,
        radius=radius,
        fill=(248, 245, 237, 255),
        outline=(219, 201, 170, 255),
        width=max(3, size // 256),
    )

    return canvas


def trim_logo(logo: Image.Image) -> Image.Image:
    rgba = logo.convert("RGBA")
    bbox = rgba.getbbox()
    if not bbox:
        return rgba
    return rgba.crop(bbox)


def zoom_center(logo: Image.Image, factor: float = 1.5) -> Image.Image:
    if factor <= 1:
        return logo
    w, h = logo.size
    crop_w = max(1, int(w / factor))
    crop_h = max(1, int(h / factor))
    left = (w - crop_w) // 2
    top = (h - crop_h) // 2
    cropped = logo.crop((left, top, left + crop_w, top + crop_h))
    return cropped.resize((w, h), Image.Resampling.LANCZOS)


def place_logo(base: Image.Image, logo: Image.Image, ratio: float = 0.86) -> Image.Image:
    logo = trim_logo(logo)
    logo = zoom_center(logo, factor=1.5)
    target = int(base.width * ratio)
    logo.thumbnail((target, target), Image.Resampling.LANCZOS)
    x = (base.width - logo.width) // 2
    y = (base.height - logo.height) // 2
    out = base.copy()
    out.alpha_composite(logo, (x, y))
    return out


def main() -> None:
    if not SRC_ICON.exists():
        raise FileNotFoundError(f"missing source icon: {SRC_ICON}")

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    logo = Image.open(SRC_ICON)

    icon = place_logo(create_icon_canvas(1024), logo, ratio=0.90)
    icon.save(OUT_ICON)

    splash = place_logo(create_icon_canvas(2732), logo, ratio=0.58)
    splash.save(OUT_SPLASH)

    print(f"generated: {OUT_ICON}")
    print(f"generated: {OUT_SPLASH}")


if __name__ == "__main__":
    main()
