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


def extract_subject(logo: Image.Image, luma_threshold: int = 225) -> Image.Image:
    rgba = logo.convert("RGBA")
    pixels = []
    for r, g, b, a in rgba.getdata():
        if a == 0:
            pixels.append((r, g, b, a))
            continue
        luma = (r + g + b) / 3
        if luma > luma_threshold:
            pixels.append((r, g, b, 0))
        else:
            pixels.append((r, g, b, a))
    out = Image.new("RGBA", rgba.size)
    out.putdata(pixels)
    return out


def place_logo(base: Image.Image, logo: Image.Image, ratio: float = 0.86) -> Image.Image:
    logo = trim_logo(logo)
    target = int(base.width * ratio)
    scale = min(target / logo.width, target / logo.height)
    resized = (
        max(1, int(round(logo.width * scale))),
        max(1, int(round(logo.height * scale))),
    )
    logo = logo.resize(resized, Image.Resampling.LANCZOS)
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

    icon = place_logo(create_icon_canvas(1024), logo, ratio=0.62)
    icon.save(OUT_ICON)

    splash = place_logo(create_icon_canvas(2732), logo, ratio=0.39)
    splash.save(OUT_SPLASH)

    print(f"generated: {OUT_ICON}")
    print(f"generated: {OUT_SPLASH}")


if __name__ == "__main__":
    main()
