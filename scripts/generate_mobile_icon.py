from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter


ROOT = Path(__file__).resolve().parents[1]
SRC_ICON = ROOT / "client" / "app-icon.png"
OUT_DIR = ROOT / "mobile" / "resources"
OUT_ICON = OUT_DIR / "icon.png"
OUT_SPLASH = OUT_DIR / "splash.png"


def create_icon_canvas(size: int = 1024) -> Image.Image:
    canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(canvas)

    margin = int(size * 0.10)
    rect = [margin, margin, size - margin, size - margin]
    radius = int(size * 0.22)

    # Ancient-scroll-like warm white rounded background
    draw.rounded_rectangle(
        rect,
        radius=radius,
        fill=(248, 245, 237, 255),
        outline=(219, 201, 170, 255),
        width=max(3, size // 256),
    )

    # Soft center highlight + edge shade to avoid flat background
    glow = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    gdraw = ImageDraw.Draw(glow)
    gdraw.ellipse(
        [margin + size * 0.08, margin + size * 0.04, size - margin - size * 0.08, size - margin - size * 0.16],
        fill=(255, 255, 255, 60),
    )
    gdraw.rounded_rectangle(
        rect,
        radius=radius,
        outline=(145, 110, 70, 50),
        width=max(2, size // 300),
    )
    glow = glow.filter(ImageFilter.GaussianBlur(radius=max(8, size // 128)))
    canvas.alpha_composite(glow)

    return canvas


def place_logo(base: Image.Image, logo: Image.Image, ratio: float = 0.58) -> Image.Image:
    logo = logo.convert("RGBA")
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

    icon = place_logo(create_icon_canvas(1024), logo, ratio=0.56)
    icon.save(OUT_ICON)

    splash = place_logo(create_icon_canvas(2732), logo, ratio=0.36)
    splash.save(OUT_SPLASH)

    print(f"generated: {OUT_ICON}")
    print(f"generated: {OUT_SPLASH}")


if __name__ == "__main__":
    main()
