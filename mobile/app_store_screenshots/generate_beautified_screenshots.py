from __future__ import annotations

import math
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont


OUT_DIR = Path(__file__).resolve().parent
W, H = 1125, 2436
FONT_PATH = "/System/Library/Fonts/Hiragino Sans GB.ttc"


class C:
    bg_top = (249, 251, 254)
    bg_bottom = (241, 245, 250)
    card = (255, 255, 255)
    text = (31, 38, 50)
    muted = (118, 128, 143)
    light = (166, 176, 190)
    line = (229, 234, 241)
    green = (25, 190, 161)
    green_dark = (16, 153, 109)
    green_soft = (226, 248, 242)
    orange = (224, 126, 0)
    orange_soft = (255, 242, 222)
    blue = (61, 134, 235)
    blue_soft = (232, 241, 255)
    red = (232, 82, 73)
    red_soft = (255, 234, 232)


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
    return ImageFont.truetype(FONT_PATH, size=size, index=2 if bold else 0)


def make_canvas() -> Image.Image:
    img = Image.new("RGB", (W, H), C.bg_top)
    px = img.load()
    for y in range(H):
        t = y / (H - 1)
        r = int(C.bg_top[0] * (1 - t) + C.bg_bottom[0] * t)
        g = int(C.bg_top[1] * (1 - t) + C.bg_bottom[1] * t)
        b = int(C.bg_top[2] * (1 - t) + C.bg_bottom[2] * t)
        for x in range(W):
            px[x, y] = (r, g, b)
    return img


def draw_shadow_card(
    img: Image.Image,
    xy: tuple[int, int, int, int],
    radius: int = 32,
    fill: tuple[int, int, int] = C.card,
    outline: tuple[int, int, int] = C.line,
    shadow_alpha: int = 22,
):
    layer = Image.new("RGBA", img.size, (0, 0, 0, 0))
    ld = ImageDraw.Draw(layer)
    x1, y1, x2, y2 = xy
    ld.rounded_rectangle(
        (x1, y1 + 10, x2, y2 + 14),
        radius=radius,
        fill=(30, 50, 80, shadow_alpha),
    )
    layer = layer.filter(ImageFilter.GaussianBlur(18))
    img.alpha_composite(layer)
    d = ImageDraw.Draw(img)
    d.rounded_rectangle(xy, radius=radius, fill=fill, outline=outline, width=2)


def draw_text(
    d: ImageDraw.ImageDraw,
    xy: tuple[int, int],
    text: str,
    size: int,
    fill: tuple[int, int, int] = C.text,
    bold: bool = False,
    anchor: str | None = None,
):
    d.text(xy, text, font=font(size, bold), fill=fill, anchor=anchor)


def text_len(d: ImageDraw.ImageDraw, text: str, f: ImageFont.FreeTypeFont) -> float:
    return d.textlength(text, font=f)


def ellipsize(d: ImageDraw.ImageDraw, text: str, f: ImageFont.FreeTypeFont, max_w: int) -> str:
    if text_len(d, text, f) <= max_w:
        return text
    s = text
    while s and text_len(d, s + "...", f) > max_w:
        s = s[:-1]
    return s + "..."


def wrap_text(
    d: ImageDraw.ImageDraw,
    text: str,
    f: ImageFont.FreeTypeFont,
    max_w: int,
    max_lines: int,
) -> list[str]:
    lines: list[str] = []
    current = ""
    for ch in text:
        probe = current + ch
        if current and text_len(d, probe, f) > max_w:
            lines.append(current)
            current = ch
            if len(lines) == max_lines:
                break
        else:
            current = probe
    if len(lines) < max_lines and current:
        lines.append(current)
    if len(lines) > max_lines:
        lines = lines[:max_lines]
    if len(lines) == max_lines and text_len(d, lines[-1], f) > max_w:
        lines[-1] = ellipsize(d, lines[-1], f, max_w)
    elif len(lines) == max_lines and len("".join(lines)) < len(text):
        lines[-1] = ellipsize(d, lines[-1], f, max_w)
    return lines


def multiline(
    d: ImageDraw.ImageDraw,
    xy: tuple[int, int],
    text: str,
    size: int,
    max_w: int,
    max_lines: int,
    fill: tuple[int, int, int] = C.muted,
    bold: bool = False,
    line_gap: int = 16,
):
    f = font(size, bold)
    x, y = xy
    for i, line in enumerate(wrap_text(d, text, f, max_w, max_lines)):
        d.text((x, y + i * (size + line_gap)), line, font=f, fill=fill)


def draw_status_bar(d: ImageDraw.ImageDraw):
    draw_text(d, (84, 54), "11:47", 44, (0, 0, 0), bold=True)
    for i in range(4):
        x = 884 + i * 14
        y = 79
        shade = 196 if i < 3 else 222
        d.ellipse((x, y, x + 8, y + 8), fill=(shade, shade, shade))
    cx, cy = 970, 72
    d.pieslice((cx - 30, cy - 30, cx + 30, cy + 30), 220, 320, fill=(0, 0, 0))
    d.pieslice((cx - 23, cy - 23, cx + 23, cy + 23), 220, 320, fill=C.bg_top)
    d.pieslice((cx - 21, cy - 21, cx + 21, cy + 21), 222, 318, fill=(0, 0, 0))
    d.pieslice((cx - 14, cy - 14, cx + 14, cy + 14), 222, 318, fill=C.bg_top)
    d.pieslice((cx - 12, cy - 12, cx + 12, cy + 12), 225, 315, fill=(0, 0, 0))
    d.ellipse((cx - 4, cy + 7, cx + 4, cy + 15), fill=(0, 0, 0))
    d.rounded_rectangle((1010, 52, 1074, 85), radius=9, outline=(110, 110, 110), width=4)
    d.rectangle((1077, 62, 1084, 75), fill=(110, 110, 110))
    d.rounded_rectangle((1016, 58, 1058, 79), radius=5, fill=(89, 220, 123))


def draw_menu(d: ImageDraw.ImageDraw):
    d.ellipse((55, 156, 187, 288), fill=(255, 255, 255))
    for y in (212, 229, 246):
        d.rounded_rectangle((93, y, 150, y + 6), radius=3, fill=C.text)


def icon_refresh(d: ImageDraw.ImageDraw, cx: int, cy: int, color=C.green, size: int = 44):
    d.arc((cx - size // 2, cy - size // 2, cx + size // 2, cy + size // 2), 35, 315, fill=color, width=7)
    d.polygon([(cx + 17, cy - 20), (cx + 33, cy - 18), (cx + 22, cy - 5)], fill=color)


def icon_check(d: ImageDraw.ImageDraw, cx: int, cy: int, color=C.green, size: int = 44):
    d.ellipse((cx - size // 2, cy - size // 2, cx + size // 2, cy + size // 2), outline=color, width=6)
    d.line((cx - 14, cy, cx - 4, cy + 10, cx + 17, cy - 15), fill=color, width=7, joint="curve")


def icon_bell(d: ImageDraw.ImageDraw, cx: int, cy: int, color=C.orange, size: int = 42):
    scale = size / 42
    w = max(4, int(6 * scale))
    d.arc((cx - 20 * scale, cy - 18 * scale, cx + 20 * scale, cy + 24 * scale), 190, 350, fill=color, width=w)
    d.line((cx - 20 * scale, cy + 7 * scale, cx - 20 * scale, cy + 22 * scale), fill=color, width=w)
    d.line((cx + 20 * scale, cy + 7 * scale, cx + 20 * scale, cy + 22 * scale), fill=color, width=w)
    d.line((cx - 25 * scale, cy + 22 * scale, cx + 25 * scale, cy + 22 * scale), fill=color, width=w)
    d.ellipse((cx - 6 * scale, cy + 28 * scale, cx + 6 * scale, cy + 40 * scale), fill=color)
    d.line((cx, cy - 26 * scale, cx, cy - 16 * scale), fill=color, width=w)


def icon_inbox(d: ImageDraw.ImageDraw, cx: int, cy: int, color=C.muted, size: int = 45):
    x1, y1, x2, y2 = cx - size // 2, cy - size // 2, cx + size // 2, cy + size // 2
    d.rounded_rectangle((x1, y1, x2, y2), radius=5, outline=color, width=5)
    d.line((x1 + 4, cy + 6, cx - 10, cy + 6, cx - 3, cy + 15, cx + 3, cy + 15, cx + 10, cy + 6, x2 - 4, cy + 6), fill=color, width=5)


def icon_doc(d: ImageDraw.ImageDraw, x: int, y: int, color=C.green_dark):
    d.rounded_rectangle((x, y, x + 48, y + 58), radius=5, outline=color, width=6)
    for yy in (y + 17, y + 31, y + 45):
        d.line((x + 13, yy, x + 35, yy), fill=color, width=5)


def icon_list_pen(d: ImageDraw.ImageDraw, cx: int, cy: int, color=C.text):
    for yy in (cy - 18, cy, cy + 18):
        d.rounded_rectangle((cx - 22, yy - 3, cx + 7, yy + 3), radius=3, fill=color)
    d.line((cx + 15, cy + 22, cx + 31, cy + 6), fill=color, width=6)
    d.polygon([(cx + 31, cy + 6), (cx + 35, cy + 2), (cx + 37, cy + 11)], fill=color)


def icon_compass(d: ImageDraw.ImageDraw, cx: int, cy: int, color=C.text):
    d.ellipse((cx - 27, cy - 27, cx + 27, cy + 27), outline=color, width=5)
    d.polygon([(cx + 14, cy - 18), (cx + 3, cy + 9), (cx - 16, cy + 18), (cx - 4, cy - 9)], fill=color)
    d.ellipse((cx - 4, cy - 4, cx + 4, cy + 4), fill=(255, 255, 255))


def icon_mic(d: ImageDraw.ImageDraw, cx: int, cy: int, color=(255, 255, 255), scale: float = 1.0):
    d.rounded_rectangle((cx - 18 * scale, cy - 48 * scale, cx + 18 * scale, cy + 18 * scale), radius=int(18 * scale), fill=color)
    d.arc((cx - 48 * scale, cy - 20 * scale, cx + 48 * scale, cy + 62 * scale), 28, 152, fill=color, width=int(8 * scale))
    d.line((cx, cy + 50 * scale, cx, cy + 76 * scale), fill=color, width=int(8 * scale))
    d.line((cx - 28 * scale, cy + 76 * scale, cx + 28 * scale, cy + 76 * scale), fill=color, width=int(8 * scale))
    d.ellipse((cx + 37 * scale, cy - 48 * scale, cx + 47 * scale, cy - 38 * scale), fill=color)


def draw_pill(
    d: ImageDraw.ImageDraw,
    xy: tuple[int, int, int, int],
    text: str,
    fill: tuple[int, int, int],
    outline: tuple[int, int, int],
    text_color: tuple[int, int, int],
    size: int = 30,
    bold: bool = True,
):
    x1, y1, x2, y2 = xy
    d.rounded_rectangle(xy, radius=(y2 - y1) // 2, fill=fill, outline=outline, width=2)
    f = font(size, bold)
    bbox = d.textbbox((0, 0), text, font=f)
    d.text(((x1 + x2 - bbox[2] + bbox[0]) / 2, (y1 + y2 - bbox[3] + bbox[1]) / 2 - 2), text, font=f, fill=text_color)


def draw_bottom_nav(img: Image.Image, active: int):
    d = ImageDraw.Draw(img)
    layer = Image.new("RGBA", img.size, (0, 0, 0, 0))
    ld = ImageDraw.Draw(layer)
    ld.rounded_rectangle((190, 2144, 740, 2268), radius=68, fill=(34, 50, 76, 28))
    layer = layer.filter(ImageFilter.GaussianBlur(18))
    img.alpha_composite(layer)
    d.rounded_rectangle((190, 2130, 740, 2255), radius=64, fill=(255, 255, 255), outline=C.line, width=2)
    d.polygon([(710, 2165), (742, 2192), (710, 2220)], fill=(255, 255, 255), outline=C.line)
    centers = [(280, 2192), (430, 2192), (580, 2192)]
    for i, (cx, cy) in enumerate(centers):
        if i == active:
            d.ellipse((cx - 63, cy - 63, cx + 63, cy + 63), fill=(0, 0, 0))
            icon_color = (255, 255, 255)
        else:
            d.ellipse((cx - 58, cy - 58, cx + 58, cy + 58), fill=(248, 249, 251))
            icon_color = C.text
        if i == 0:
            icon_list_pen(d, cx, cy, icon_color)
        elif i == 1:
            icon_compass(d, cx, cy, icon_color)
        else:
            icon_bell(d, cx, cy - 8, icon_color, 36)
    glow = Image.new("RGBA", img.size, (0, 0, 0, 0))
    gd = ImageDraw.Draw(glow)
    gd.ellipse((752, 2074, 1030, 2352), fill=(81, 235, 153, 72))
    gd.ellipse((790, 2112, 992, 2314), fill=(81, 235, 153, 95))
    glow = glow.filter(ImageFilter.GaussianBlur(34))
    img.alpha_composite(glow)
    d.ellipse((770, 2100, 1012, 2342), fill=(182, 250, 210), outline=(199, 238, 220), width=2)
    d.ellipse((810, 2140, 972, 2302), fill=(76, 224, 128))
    icon_mic(d, 891, 2213, (255, 255, 255), 0.82)
    d.line((368, 2384, 757, 2384), fill=(0, 0, 0), width=13)


def stat_block(d: ImageDraw.ImageDraw, cx: int, top: int, icon: str, number: str, label: str, color: tuple[int, int, int]):
    if icon == "bell":
        icon_bell(d, cx, top + 32, color, 42)
    elif icon == "check":
        icon_check(d, cx, top + 35, color, 42)
    else:
        icon_inbox(d, cx, top + 35, color, 42)
    draw_text(d, (cx, top + 83), number, 62, C.text, bold=True, anchor="ma")
    draw_text(d, (cx, top + 152), label, 35, C.light, bold=True, anchor="ma")


def service_card(
    img: Image.Image,
    y: int,
    title: str,
    body: str,
    step: str,
    meta: str,
    status: str,
    status_color: tuple[int, int, int],
):
    d = ImageDraw.Draw(img)
    draw_shadow_card(img, (56, y, 1069, y + 340), radius=27)
    draw_text(d, (96, y + 42), ellipsize(d, title, font(41, True), 730), 41, C.text, bold=True)
    draw_pill(d, (902, y + 36, 1016, y + 84), status, (248, 250, 252), C.line, status_color, size=28)
    multiline(d, (96, y + 108), body, 34, 880, 2, C.muted, bold=True, line_gap=14)
    draw_text(d, (96, y + 218), "下一步：", 34, C.text, bold=True)
    draw_text(d, (236, y + 218), ellipsize(d, step, font(34, True), 720), 34, C.muted, bold=True)
    draw_text(d, (96, y + 282), ellipsize(d, meta, font(29), 825), 29, C.light)
    d.line((1010, y + 282, 1025, y + 299, 1010, y + 316), fill=C.light, width=5)


def screen_service() -> Image.Image:
    img = make_canvas().convert("RGBA")
    d = ImageDraw.Draw(img)
    draw_status_bar(d)
    draw_menu(d)
    draw_text(d, (56, 338), "服务提醒", 59, C.text, bold=True)
    multiline(d, (56, 420), "AI 自动梳理今日要服务谁、为什么提醒和下一步动作。", 41, 780, 2, C.muted, bold=True, line_gap=18)
    icon_doc(d, 828, 391, C.muted)
    icon_refresh(d, 1005, 418, C.muted, 55)

    draw_shadow_card(img, (56, 565, 1069, 832), radius=33)
    for x in (407, 713):
        d.line((x, 640, x, 770), fill=C.line, width=3)
    stat_block(d, 255, 615, "bell", "12", "待处理", C.orange)
    stat_block(d, 560, 615, "check", "38", "已完成", C.green)
    stat_block(d, 865, 615, "inbox", "50", "全部", C.muted)

    service_card(
        img,
        900,
        "开放日家长邀约与接待确认",
        "本周六开放日已预约 32 位家长，3 组需确认到场时间和试听需求。",
        "优先联系未确认家庭，发送到园路线与课程安排",
        "招生增长 · 高意向 · 今日 10:30 · 负责人：张老师",
        "待处理",
        C.orange,
    )
    service_card(
        img,
        1288,
        "试听课后黄金 24 小时跟进",
        "小明妈妈昨天完成试听课，系统识别为高意向客户，关注课程效果与接送便利。",
        "发送个性化反馈，并预约下次沟通时间",
        "客户跟进 · 高优先级 · 已生成 2 条话术",
        "高优先",
        C.green_dark,
    )

    draw_shadow_card(img, (56, 1688, 1069, 1948), radius=30, fill=(252, 255, 253), outline=(218, 242, 235), shadow_alpha=18)
    draw_text(d, (96, 1727), "本周服务进展", 39, C.text, bold=True)
    metrics = [("86%", "响应率"), ("12", "新增线索"), ("7", "已预约试听")]
    for i, (num, lab) in enumerate(metrics):
        cx = 235 + i * 310
        d.rounded_rectangle((cx - 102, 1798, cx + 102, 1902), radius=30, fill=C.green_soft)
        draw_text(d, (cx, 1812), num, 43, C.green_dark, bold=True, anchor="ma")
        draw_text(d, (cx, 1862), lab, 25, C.green_dark, bold=True, anchor="ma")

    draw_bottom_nav(img, active=2)
    return img.convert("RGB")


def knowledge_tile(
    img: Image.Image,
    xy: tuple[int, int, int, int],
    title: str,
    desc: str,
    count: str,
    accent: tuple[int, int, int],
    soft: tuple[int, int, int],
):
    d = ImageDraw.Draw(img)
    draw_shadow_card(img, xy, radius=34, shadow_alpha=18)
    x1, y1, x2, y2 = xy
    d.rounded_rectangle((x1 + 40, y1 + 38, x1 + 118, y1 + 116), radius=22, fill=soft)
    icon_doc(d, x1 + 55, y1 + 49, accent)
    draw_text(d, (x1 + 142, y1 + 55), title, 42, C.text, bold=True)
    multiline(d, (x1 + 40, y1 + 145), desc, 33, x2 - x1 - 80, 2, C.muted, bold=True, line_gap=12)
    draw_text(d, (x1 + 40, y2 - 86), count, 37, accent, bold=True)


def memory_card(
    img: Image.Image,
    y: int,
    title: str,
    body: str,
    date: str,
    badge: str,
    action: str,
):
    d = ImageDraw.Draw(img)
    draw_shadow_card(img, (56, y, 1069, y + 432), radius=34)
    draw_text(d, (104, y + 62), ellipsize(d, title, font(43, True), 830), 43, C.text, bold=True)
    multiline(d, (104, y + 134), body, 37, 888, 3, C.muted, bold=True, line_gap=12)
    draw_text(d, (104, y + 338), date, 34, C.light, bold=True)
    draw_pill(d, (380, y + 321, 594, y + 382), badge, C.green_soft, (179, 233, 215), C.green_dark, size=30)
    draw_pill(d, (624, y + 321, 880, y + 382), action, (226, 247, 244), (226, 247, 244), C.green, size=31)
    for yy in (y + 332, y + 353, y + 374):
        d.ellipse((970, yy, 981, yy + 11), fill=C.light)


def screen_knowledge() -> Image.Image:
    img = make_canvas().convert("RGBA")
    d = ImageDraw.Draw(img)
    draw_status_bar(d)
    draw_menu(d)
    draw_text(d, (56, 352), "知识库", 59, C.text, bold=True)
    d.rounded_rectangle((842, 321, 1069, 435), radius=57, fill=(255, 255, 255), outline=C.line, width=2)
    draw_text(d, (887, 357), "更多", 39, C.muted, bold=True)
    d.line((995, 367, 1014, 386, 995, 405), fill=C.muted, width=5)

    knowledge_tile(img, (56, 500, 542, 842), "招生话术", "试听邀约、异议处理、续费跟进", "128 个内容", C.green_dark, C.green_soft)
    knowledge_tile(img, (584, 500, 1069, 842), "运营支持", "园所 SOP、活动清单、教师培训", "64 个流程", C.blue, C.blue_soft)

    draw_text(d, (56, 980), "全部记忆", 61, C.text, bold=True)
    d.line((304, 1008, 331, 1035, 358, 1008), fill=C.text, width=6)
    d.rounded_rectangle((126, 1080, 217, 1094), radius=7, fill=C.text)

    memory_card(
        img,
        1146,
        "开放日家长沟通纪要",
        "AI 已整理 18 条家长问题，提取出价格、接送、课程效果三类关注点，并生成跟进建议。",
        "9月5日 11:15",
        "转写成功",
        "提取 6 条服务",
    )
    memory_card(
        img,
        1624,
        "试听课后跟进录音",
        "录音 12 分钟已完成转写，自动总结孩子课堂表现、家长疑虑和下次邀约时机。",
        "9月4日 16:28",
        "已生成摘要",
        "提取 4 条服务",
    )
    draw_bottom_nav(img, active=0)
    return img.convert("RGB")


def topic_card(
    img: Image.Image,
    y: int,
    title: str,
    body: str,
    meta: str,
    tag: str,
    accent: tuple[int, int, int],
    soft: tuple[int, int, int],
    h: int = 290,
):
    d = ImageDraw.Draw(img)
    draw_shadow_card(img, (56, y, 1069, y + h), radius=24, shadow_alpha=16)
    d.rounded_rectangle((88, y + 32, 278, y + 222), radius=22, fill=soft)
    icon_doc(d, 157, y + 78, accent)
    draw_text(d, (144, y + 153), tag, 30, accent, bold=True)
    draw_text(d, (312, y + 42), ellipsize(d, title, font(39, True), 690), 39, C.text, bold=True)
    multiline(d, (312, y + 100), body, 33, 690, 2, C.muted, bold=True, line_gap=14)
    draw_text(d, (312, y + h - 68), meta, 30, C.light, bold=True)


def draw_category_pill(
    d: ImageDraw.ImageDraw,
    x: int,
    y: int,
    w: int,
    text: str,
    active: bool = False,
):
    if active:
        d.rounded_rectangle((x, y, x + w, y + 82), radius=41, fill=(235, 253, 248), outline=(174, 238, 219), width=3)
        draw_text(d, (x + w // 2, y + 20), text, 34, C.green_dark, bold=True, anchor="ma")
    else:
        d.rounded_rectangle((x, y, x + w, y + 82), radius=41, fill=(248, 250, 252), outline=C.line, width=2)
        draw_text(d, (x + w // 2, y + 20), text, 34, C.muted, bold=True, anchor="ma")


def screen_topics() -> Image.Image:
    img = make_canvas().convert("RGBA")
    d = ImageDraw.Draw(img)
    draw_status_bar(d)
    draw_menu(d)
    draw_text(d, (56, 340), "精选主题", 56, C.text, bold=True)
    icon_refresh(d, 884, 372, C.green, 39)
    draw_text(d, (936, 345), "换一批", 36, C.green, bold=True)

    topic_card(
        img,
        438,
        "家长犹豫不报名？4 类顾虑逐个击破",
        "识别价格、信任、接送、家庭决策四类顾虑，自动生成针对性跟进策略。",
        "9月5日 11:21  @创作者",
        "招生",
        C.green_dark,
        (213, 238, 226),
        h=295,
    )
    topic_card(
        img,
        774,
        "开放日高转化：从展示到成交的 5 步 SOP",
        "优化入园接待、儿童体验、透明答疑和会后追踪，降低家长决策阻力。",
        "9月5日 11:20  @创作者",
        "运营",
        C.blue,
        C.blue_soft,
        h=295,
    )

    draw_category_pill(d, 56, 1132, 184, "推荐 18", True)
    draw_category_pill(d, 262, 1132, 250, "招生增长 12")
    draw_category_pill(d, 536, 1132, 250, "家长服务 9")
    draw_category_pill(d, 810, 1132, 260, "活动策划 6")
    draw_text(d, (56, 1290), "推荐", 52, C.text, bold=True)
    draw_text(d, (998, 1300), "18 条", 37, C.light, bold=True, anchor="ra")

    topic_card(
        img,
        1380,
        "老生续费提醒：提前 30 天建立信任",
        "按孩子成长记录、课堂反馈和家长关注点，拆解续费沟通节奏。",
        "6 个步骤 · 3 条话术 · 适合班主任",
        "续费",
        C.orange,
        C.orange_soft,
        h=280,
    )
    topic_card(
        img,
        1704,
        "朋友圈招生文案：一周 7 条可直接发",
        "从真实课堂、孩子变化、家长反馈中提炼内容，持续触达潜在客户。",
        "7 条模板 · 支持复制 · 今日更新",
        "文案",
        C.green_dark,
        C.green_soft,
        h=280,
    )
    draw_bottom_nav(img, active=1)
    return img.convert("RGB")


def main() -> None:
    outputs = [
        ("01-service-reminders-beautified.png", screen_service()),
        ("02-knowledge-base-beautified.png", screen_knowledge()),
        ("03-featured-topics-beautified.png", screen_topics()),
    ]
    for name, image in outputs:
        path = OUT_DIR / name
        image.save(path, "PNG", optimize=True)
        print(path)


if __name__ == "__main__":
    main()
