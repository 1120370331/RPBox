#!/usr/bin/env python3
import json
import os
from collections import deque
from glob import glob

from PIL import Image


def is_near_black(pixel, threshold):
    r, g, b, a = pixel
    if a == 0:
        return False
    return r <= threshold and g <= threshold and b <= threshold


def remove_black_background(img, threshold):
    width, height = img.size
    pixels = img.load()
    visited = [False] * (width * height)
    queue = deque()

    def try_push(x, y):
        idx = y * width + x
        if visited[idx]:
            return
        if not is_near_black(pixels[x, y], threshold):
            return
        visited[idx] = True
        queue.append((x, y))

    # Flood fill from edges to avoid wiping dark details inside the emote.
    for x in range(width):
        try_push(x, 0)
        if height > 1:
            try_push(x, height - 1)
    for y in range(height):
        try_push(0, y)
        if width > 1:
            try_push(width - 1, y)

    while queue:
        x, y = queue.popleft()
        if x > 0:
            try_push(x - 1, y)
        if x < width - 1:
            try_push(x + 1, y)
        if y > 0:
            try_push(x, y - 1)
        if y < height - 1:
            try_push(x, y + 1)

    for y in range(height):
        row_offset = y * width
        for x in range(width):
            if visited[row_offset + x]:
                r, g, b, _ = pixels[x, y]
                pixels[x, y] = (r, g, b, 0)

    return img


def load_configs(config_dir):
    configs = []
    for path in sorted(glob(os.path.join(config_dir, "*.json"))):
        with open(path, "r", encoding="utf-8") as f:
            configs.append(json.load(f))
    return configs


def process_pack(pack):
    pack_id = pack["id"]
    name = pack["name"]
    source_image = pack["source_image"]
    grid = pack["grid"]
    rows = int(grid["rows"])
    cols = int(grid["cols"])
    output_dir = pack.get("output_dir") or os.path.join("server", "storage", "emotes", pack_id)
    size = int(pack.get("size", 128))
    threshold = int(pack.get("background_threshold", 5))

    sheet = Image.open(source_image).convert("RGBA")
    tile_width = sheet.width // cols
    tile_height = sheet.height // rows

    os.makedirs(output_dir, exist_ok=True)

    items_out = []
    items = pack.get("items", [])
    for idx, item in enumerate(items):
        item_id = item["id"]
        index = int(item.get("index", idx + 1))
        row = (index - 1) // cols
        col = (index - 1) % cols
        left = col * tile_width
        upper = row * tile_height
        right = left + tile_width
        lower = upper + tile_height

        tile = sheet.crop((left, upper, right, lower))
        tile = remove_black_background(tile, threshold)
        tile = tile.resize((size, size), Image.LANCZOS)

        output_path = os.path.join(output_dir, f"{item_id}.png")
        tile.save(output_path, "PNG", optimize=True)

        items_out.append({
            "id": item_id,
            "name": item["name"],
            "text": item.get("text", ""),
            "file": f"/emotes/{pack_id}/{item_id}.png",
        })

    icon_id = pack.get("icon_id") or (items[0]["id"] if items else "")
    pack_out = {
        "id": pack_id,
        "name": name,
        "icon": f"/emotes/{pack_id}/{icon_id}.png" if icon_id else "",
        "items": items_out,
    }
    return pack_out


def main():
    config_dir = os.path.join("server", "storage", "emotes", "packs")
    manifest_path = os.path.join("server", "storage", "emotes", "manifest.json")

    packs = load_configs(config_dir)
    if not packs:
        raise SystemExit("No emote pack configs found.")

    manifest = {"packs": []}
    for pack in packs:
        manifest["packs"].append(process_pack(pack))

    with open(manifest_path, "w", encoding="utf-8") as f:
        json.dump(manifest, f, ensure_ascii=False, indent=2)
        f.write("\n")


if __name__ == "__main__":
    main()
