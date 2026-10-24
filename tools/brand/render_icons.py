"""Renders the Godwit icon (assets/brand/godwit-icon.svg) into the app's icon files.

Usage, from the repository root:  python tools/brand/render_icons.py
Needs Google Chrome (to draw the SVG) and Pillow (pip install pillow).
"""
import pathlib
import subprocess
import tempfile

from PIL import Image

ROOT = pathlib.Path(__file__).resolve().parents[2]
ICON_SVG = ROOT / "assets/brand/godwit-icon.svg"
CHROME = r"C:\Program Files\Google\Chrome\Application\chrome.exe"


def render(svg: pathlib.Path, size: int, out: pathlib.Path) -> Image.Image:
    """Draws the SVG at size x size with a transparent background."""
    with tempfile.TemporaryDirectory() as tmp:
        page = pathlib.Path(tmp) / "page.html"
        page.write_text(
            f'<!doctype html><body style="margin:0;background:transparent">'
            f'<img src="{svg.as_uri()}" width="{size}" height="{size}" style="display:block">',
            encoding="utf-8",
        )
        subprocess.run(
            [CHROME, "--headless=new", "--disable-gpu", "--hide-scrollbars", "--force-device-scale-factor=1",
             "--default-background-color=00000000", f"--user-data-dir={tmp}", f"--window-size={size},{size}",
             f"--screenshot={out}", page.as_uri()],
            check=True, capture_output=True, timeout=60,
        )
    return Image.open(out).convert("RGBA")


def main() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        big = render(ICON_SVG, 1024, pathlib.Path(tmp) / "icon_1024.png")

    # Windows: one .ico with the sizes Explorer and the taskbar use
    ico = ROOT / "app/windows/runner/resources/app_icon.ico"
    big.save(ico, sizes=[(s, s) for s in (16, 20, 24, 32, 40, 48, 64, 96, 128, 256)])

    # macOS: the tile with the standard margin around it
    mac = ROOT / "app/macos/Runner/Assets.xcassets/AppIcon.appiconset"
    for size in (16, 32, 64, 128, 256, 512, 1024):
        canvas = Image.new("RGBA", (size, size))
        inner = round(size * 0.8)
        tile = big.resize((inner, inner), Image.LANCZOS)
        canvas.alpha_composite(tile, ((size - inner) // 2, (size - inner) // 2))
        canvas.save(mac / f"app_icon_{size}.png")

    # The brand mark in the app's navigation
    brand = ROOT / "app/assets/brand/icon.png"
    brand.parent.mkdir(parents=True, exist_ok=True)
    big.resize((256, 256), Image.LANCZOS).save(brand)

    print("Wrote", ico.relative_to(ROOT), "and the macOS and in-app icons")


if __name__ == "__main__":
    main()
