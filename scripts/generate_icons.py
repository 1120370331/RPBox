#!/usr/bin/env python3
"""
RPBox Logo 处理脚本
- 去除黑色背景（转为透明）
- 生成各种尺寸的图标
- 应用到 Tauri 项目
"""

from PIL import Image
import numpy as np
import os

# 路径配置
SOURCE_IMAGE = r"C:\Users\rog\WorkSpace\projects\github_repo\RPBox\style-demos\27a931f6-74cb-4947-966b-8c13c1391b03.png"
ICONS_DIR = r"C:\Users\rog\WorkSpace\projects\github_repo\RPBox\client\src-tauri\icons"
CLIENT_DIR = r"C:\Users\rog\WorkSpace\projects\github_repo\RPBox\client"

def remove_black_background(img, threshold=30):
    """去除黑色背景，转为透明"""
    # 转换为 RGBA
    img = img.convert("RGBA")
    data = np.array(img)

    # 找出接近黑色的像素（RGB 值都小于阈值）
    r, g, b, a = data[:,:,0], data[:,:,1], data[:,:,2], data[:,:,3]
    black_mask = (r < threshold) & (g < threshold) & (b < threshold)

    # 将黑色像素设为透明
    data[black_mask] = [0, 0, 0, 0]

    return Image.fromarray(data)

def resize_icon(img, size):
    """调整图标尺寸，保持高质量"""
    return img.resize((size, size), Image.Resampling.LANCZOS)

def save_ico(img, path, sizes=[16, 32, 48, 64, 128, 256]):
    """保存为 ICO 格式"""
    icons = [resize_icon(img, s) for s in sizes]
    icons[0].save(path, format='ICO', sizes=[(s, s) for s in sizes], append_images=icons[1:])

def main():
    print("正在加载源图像...")
    original = Image.open(SOURCE_IMAGE)
    print(f"原始尺寸: {original.size}")

    print("正在去除黑色背景...")
    img = remove_black_background(original)

    # 保存透明背景版本
    transparent_path = os.path.join(CLIENT_DIR, "app-icon.png")
    img.save(transparent_path)
    print(f"已保存透明背景版本: {transparent_path}")

    # 定义需要生成的图标尺寸
    standard_icons = {
        "32x32.png": 32,
        "64x64.png": 64,
        "128x128.png": 128,
        "128x128@2x.png": 256,
        "icon.png": 512,
    }

    square_icons = {
        "StoreLogo.png": 50,
        "Square30x30Logo.png": 30,
        "Square44x44Logo.png": 44,
        "Square71x71Logo.png": 71,
        "Square89x89Logo.png": 89,
        "Square107x107Logo.png": 107,
        "Square142x142Logo.png": 142,
        "Square150x150Logo.png": 150,
        "Square284x284Logo.png": 284,
        "Square310x310Logo.png": 310,
    }

    # 生成标准图标
    print("\n生成标准图标...")
    for name, size in standard_icons.items():
        icon = resize_icon(img, size)
        path = os.path.join(ICONS_DIR, name)
        icon.save(path)
        print(f"  {name} ({size}x{size})")

    # 生成 Square 图标
    print("\n生成 Square 图标...")
    for name, size in square_icons.items():
        icon = resize_icon(img, size)
        path = os.path.join(ICONS_DIR, name)
        icon.save(path)
        print(f"  {name} ({size}x{size})")

    # 生成 ICO 文件
    print("\n生成 ICO 文件...")
    ico_path = os.path.join(ICONS_DIR, "icon.ico")
    save_ico(img, ico_path)
    print(f"  icon.ico")

    # 复制到 release 目录
    release_ico = os.path.join(ICONS_DIR, "..", "target", "release", "resources", "icon.ico")
    if os.path.exists(os.path.dirname(release_ico)):
        save_ico(img, release_ico)
        print(f"  target/release/resources/icon.ico")

    # 生成 iOS 图标
    ios_dir = os.path.join(ICONS_DIR, "ios")
    if os.path.exists(ios_dir):
        print("\n生成 iOS 图标...")
        ios_icons = {
            "AppIcon-20x20@1x.png": 20,
            "AppIcon-20x20@2x.png": 40,
            "AppIcon-20x20@2x-1.png": 40,
            "AppIcon-20x20@3x.png": 60,
            "AppIcon-29x29@1x.png": 29,
            "AppIcon-29x29@2x.png": 58,
            "AppIcon-29x29@2x-1.png": 58,
            "AppIcon-29x29@3x.png": 87,
            "AppIcon-40x40@1x.png": 40,
            "AppIcon-40x40@2x.png": 80,
            "AppIcon-40x40@2x-1.png": 80,
            "AppIcon-40x40@3x.png": 120,
            "AppIcon-60x60@2x.png": 120,
            "AppIcon-60x60@3x.png": 180,
            "AppIcon-76x76@1x.png": 76,
            "AppIcon-76x76@2x.png": 152,
            "AppIcon-83.5x83.5@2x.png": 167,
            "AppIcon-512@2x.png": 1024,
        }
        for name, size in ios_icons.items():
            icon = resize_icon(img, size)
            path = os.path.join(ios_dir, name)
            icon.save(path)
            print(f"  {name} ({size}x{size})")

    # 生成 Android 图标
    android_dir = os.path.join(ICONS_DIR, "android")
    if os.path.exists(android_dir):
        print("\n生成 Android 图标...")
        android_sizes = {
            "mipmap-mdpi": 48,
            "mipmap-hdpi": 72,
            "mipmap-xhdpi": 96,
            "mipmap-xxhdpi": 144,
            "mipmap-xxxhdpi": 192,
        }
        for folder, size in android_sizes.items():
            folder_path = os.path.join(android_dir, folder)
            if os.path.exists(folder_path):
                icon = resize_icon(img, size)
                # ic_launcher.png
                icon.save(os.path.join(folder_path, "ic_launcher.png"))
                # ic_launcher_round.png
                icon.save(os.path.join(folder_path, "ic_launcher_round.png"))
                # ic_launcher_foreground.png (稍大一些)
                fg_size = int(size * 1.5)
                fg_icon = resize_icon(img, fg_size)
                fg_icon.save(os.path.join(folder_path, "ic_launcher_foreground.png"))
                print(f"  {folder}/ ({size}x{size})")

    print("\n图标生成完成!")

if __name__ == "__main__":
    main()
