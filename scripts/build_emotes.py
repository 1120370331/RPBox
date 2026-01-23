#!/usr/bin/env python3
import io
import json
import os
from collections import deque
from glob import glob

from PIL import Image, ImageFilter
import numpy as np

try:
    from rembg import remove as rembg_remove
except Exception:
    rembg_remove = None


def is_near_black(pixel, threshold):
    r, g, b, a = pixel
    if a == 0:
        return False
    return r <= threshold and g <= threshold and b <= threshold


def remove_black_background(img, threshold, edge_feather=2, decontam_threshold=0.98):
    width, height = img.size
    pixels = img.load()
    total = width * height
    visited = [False] * total
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

    if edge_feather <= 0:
        return img

    # Compute distance to background for edge detection.
    dist = [-1] * total
    queue.clear()
    for idx in range(total):
        if visited[idx]:
            dist[idx] = 0
            queue.append((idx % width, idx // width))

    while queue:
        x, y = queue.popleft()
        base = dist[y * width + x]
        if x > 0:
            idx = y * width + (x - 1)
            if dist[idx] == -1:
                dist[idx] = base + 1
                queue.append((x - 1, y))
        if x < width - 1:
            idx = y * width + (x + 1)
            if dist[idx] == -1:
                dist[idx] = base + 1
                queue.append((x + 1, y))
        if y > 0:
            idx = (y - 1) * width + x
            if dist[idx] == -1:
                dist[idx] = base + 1
                queue.append((x, y - 1))
        if y < height - 1:
            idx = (y + 1) * width + x
            if dist[idx] == -1:
                dist[idx] = base + 1
                queue.append((x, y + 1))

    # Propagate nearest solid colors to decontaminate black-matted edges.
    nearest = [None] * total
    queue.clear()
    for idx in range(total):
        if visited[idx]:
            continue
        if dist[idx] > edge_feather:
            x = idx % width
            y = idx // width
            nearest[idx] = pixels[x, y][:3]
            queue.append((x, y))

    if not queue:
        return img

    while queue:
        x, y = queue.popleft()
        color = nearest[y * width + x]
        if x > 0:
            idx = y * width + (x - 1)
            if not visited[idx] and nearest[idx] is None:
                nearest[idx] = color
                queue.append((x - 1, y))
        if x < width - 1:
            idx = y * width + (x + 1)
            if not visited[idx] and nearest[idx] is None:
                nearest[idx] = color
                queue.append((x + 1, y))
        if y > 0:
            idx = (y - 1) * width + x
            if not visited[idx] and nearest[idx] is None:
                nearest[idx] = color
                queue.append((x, y - 1))
        if y < height - 1:
            idx = (y + 1) * width + x
            if not visited[idx] and nearest[idx] is None:
                nearest[idx] = color
                queue.append((x, y + 1))

    min_alpha = 0.05
    for idx in range(total):
        if visited[idx]:
            continue
        if dist[idx] <= 0 or dist[idx] > edge_feather:
            continue
        target = nearest[idx]
        if target is None:
            continue
        x = idx % width
        y = idx // width
        r, g, b, _ = pixels[x, y]
        tr, tg, tb = target
        ratios = []
        if tr > 0:
            ratios.append(r / tr)
        if tg > 0:
            ratios.append(g / tg)
        if tb > 0:
            ratios.append(b / tb)
        if not ratios:
            continue
        alpha = max(ratios)
        if alpha >= decontam_threshold:
            continue
        if alpha < min_alpha:
            alpha = min_alpha
        if alpha > 1.0:
            alpha = 1.0
        new_r = min(255, int(round(r / alpha))) if r > 0 else 0
        new_g = min(255, int(round(g / alpha))) if g > 0 else 0
        new_b = min(255, int(round(b / alpha))) if b > 0 else 0
        pixels[x, y] = (new_r, new_g, new_b, int(round(alpha * 255)))

    return img


def remove_background_rembg(
    img,
    alpha_matting,
    fg_threshold,
    bg_threshold,
    erode_size,
    only_mask=False,
    post_process_mask=False,
):
    if rembg_remove is None:
        raise RuntimeError("rembg is not installed. Install it or switch background_mode.")
    buffer = io.BytesIO()
    img.save(buffer, format="PNG")
    result = rembg_remove(
        buffer.getvalue(),
        alpha_matting=alpha_matting,
        alpha_matting_foreground_threshold=fg_threshold,
        alpha_matting_background_threshold=bg_threshold,
        alpha_matting_erode_size=erode_size,
        only_mask=only_mask,
        post_process_mask=post_process_mask,
    )
    if isinstance(result, Image.Image):
        return result.convert("RGBA")
    if only_mask:
        return Image.open(io.BytesIO(result)).convert("L")
    return Image.open(io.BytesIO(result)).convert("RGBA")


def apply_alpha_mask(img, mask):
    if img.mode != "RGBA":
        img = img.convert("RGBA")
    if mask.mode != "L":
        mask = mask.convert("L")
    masked = img.copy()
    masked.putalpha(mask)
    return masked


def dematte_black(img, min_alpha=0.05):
    if img.mode != "RGBA":
        img = img.convert("RGBA")
    data = np.array(img).astype(np.float32)
    alpha = data[:, :, 3:4] / 255.0
    mask = (alpha > 0) & (alpha < 1)
    safe_alpha = np.clip(alpha, min_alpha, 1.0)
    data[:, :, :3] = np.where(mask, data[:, :, :3] / safe_alpha, data[:, :, :3])
    data = np.clip(data, 0, 255).astype(np.uint8)
    return Image.fromarray(data, "RGBA")


def drop_dark_fringe(img, color_threshold=16, alpha_threshold=200):
    if img.mode != "RGBA":
        img = img.convert("RGBA")
    data = np.array(img)
    dark = (data[:, :, 0] < color_threshold) & (data[:, :, 1] < color_threshold) & (data[:, :, 2] < color_threshold)
    fringe = dark & (data[:, :, 3] < alpha_threshold)
    data[fringe, 3] = 0
    return Image.fromarray(data, "RGBA")


def clear_black_edge_pixels(img, color_threshold=30, alpha_neighbor_threshold=10):
    if img.mode != "RGBA":
        img = img.convert("RGBA")
    data = np.array(img)
    alpha = data[:, :, 3]
    near_black = (
        (data[:, :, 0] <= color_threshold)
        & (data[:, :, 1] <= color_threshold)
        & (data[:, :, 2] <= color_threshold)
    )
    transparent = alpha <= alpha_neighbor_threshold
    pad = np.pad(transparent, 1, constant_values=True)
    adjacent_transparent = (
        pad[0:-2, 1:-1] | pad[2:, 1:-1] | pad[1:-1, 0:-2] | pad[1:-1, 2:]
        | pad[0:-2, 0:-2] | pad[0:-2, 2:] | pad[2:, 0:-2] | pad[2:, 2:]
    )
    mask = near_black & (alpha > 0) & adjacent_transparent
    if mask.any():
        data[mask, 3] = 0
    return Image.fromarray(data, "RGBA")


def resize_with_alpha(img, size):
    if img.mode != "RGBA":
        img = img.convert("RGBA")
    data = np.array(img).astype(np.float32)
    alpha = data[:, :, 3:4] / 255.0
    data[:, :, :3] *= alpha
    premult = Image.fromarray(data.astype(np.uint8), "RGBA")
    premult = premult.resize(size, Image.LANCZOS)
    data = np.array(premult).astype(np.float32)
    alpha = data[:, :, 3:4] / 255.0
    with np.errstate(divide="ignore", invalid="ignore"):
        data[:, :, :3] = np.where(alpha > 0, data[:, :, :3] / alpha, 0)
    data = np.clip(data, 0, 255).astype(np.uint8)
    return Image.fromarray(data, "RGBA")


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
    edge_feather = int(pack.get("edge_feather", 2))
    decontam_threshold = float(pack.get("decontam_threshold", 0.98))
    background_mode = pack.get("background_mode", "edge")
    apply_dematte = bool(pack.get("dematte_black", False))
    dematte_min_alpha = float(pack.get("dematte_min_alpha", 0.05))
    cleanup_fringe = bool(pack.get("cleanup_dark_fringe", False))
    fringe_color_threshold = int(pack.get("fringe_color_threshold", 16))
    fringe_alpha_threshold = int(pack.get("fringe_alpha_threshold", 200))
    cleanup_edge_black = bool(pack.get("cleanup_edge_black", False))
    edge_black_threshold = int(pack.get("edge_black_threshold", 30))
    edge_alpha_neighbor = int(pack.get("edge_alpha_neighbor_threshold", 10))
    rembg_alpha_matting = bool(pack.get("rembg_alpha_matting", True))
    rembg_fg_threshold = int(pack.get("rembg_foreground_threshold", 240))
    rembg_bg_threshold = int(pack.get("rembg_background_threshold", 10))
    rembg_erode_size = int(pack.get("rembg_erode_size", 10))
    rembg_post_process_mask = bool(pack.get("rembg_post_process_mask", True))
    rembg_mask_erode = int(pack.get("rembg_mask_erode", 0))

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
        if background_mode == "rembg":
            tile = remove_background_rembg(
                tile,
                alpha_matting=rembg_alpha_matting,
                fg_threshold=rembg_fg_threshold,
                bg_threshold=rembg_bg_threshold,
                erode_size=rembg_erode_size,
            )
        elif background_mode == "rembg_mask":
            mask = remove_background_rembg(
                tile,
                alpha_matting=rembg_alpha_matting,
                fg_threshold=rembg_fg_threshold,
                bg_threshold=rembg_bg_threshold,
                erode_size=rembg_erode_size,
                only_mask=True,
                post_process_mask=rembg_post_process_mask,
            )
            if rembg_mask_erode > 0:
                mask_erode_size = rembg_mask_erode if rembg_mask_erode % 2 == 1 else rembg_mask_erode + 1
                mask = mask.filter(ImageFilter.MinFilter(mask_erode_size))
            tile = apply_alpha_mask(tile, mask)
        elif background_mode == "edge":
            tile = remove_black_background(
                tile,
                threshold,
                edge_feather=edge_feather,
                decontam_threshold=decontam_threshold,
            )
        elif background_mode != "none":
            raise ValueError(f"Unsupported background_mode: {background_mode}")

        if apply_dematte:
            tile = dematte_black(tile, min_alpha=dematte_min_alpha)
        if cleanup_fringe:
            tile = drop_dark_fringe(
                tile,
                color_threshold=fringe_color_threshold,
                alpha_threshold=fringe_alpha_threshold,
            )
        if cleanup_edge_black:
            tile = clear_black_edge_pixels(
                tile,
                color_threshold=edge_black_threshold,
                alpha_neighbor_threshold=edge_alpha_neighbor,
            )

        tile = resize_with_alpha(tile, (size, size))

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
