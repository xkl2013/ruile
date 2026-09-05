from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter

from generate_beautified_screenshots import (
    C,
    draw_pill,
    draw_shadow_card,
    draw_text,
    ellipsize,
    font,
    icon_bell,
    icon_check,
    icon_compass,
    icon_doc,
    icon_inbox,
    icon_list_pen,
    icon_mic,
    icon_refresh,
    multiline,
    text_len,
)


OUT_DIR = Path(__file__).resolve().parent / "ipad-13"
W, H = 2064, 2752


def canvas() -> Image.Image:
    img = Image.new("RGB", (W, H), C.bg_top)
    px = img.load()
    for y in range(H):
        t = y / (H - 1)
        color = tuple(int(C.bg_top[i] * (1 - t) + C.bg_bottom[i] * t) for i in range(3))
        for x in range(W):
            px[x, y] = color
    return img.convert("RGBA")


def status_bar(d: ImageDraw.ImageDraw):
    draw_text(d, (84, 54), "11:47", 44, (0, 0, 0), bold=True)
    for i in range(4):
        d.ellipse((1810 + i * 16, 76, 1820 + i * 16, 86), fill=(196, 196, 196))
    cx, cy = 1935, 72
    d.pieslice((cx - 32, cy - 32, cx + 32, cy + 32), 220, 320, fill=(0, 0, 0))
    d.pieslice((cx - 24, cy - 24, cx + 24, cy + 24), 220, 320, fill=C.bg_top)
    d.pieslice((cx - 21, cy - 21, cx + 21, cy + 21), 222, 318, fill=(0, 0, 0))
    d.pieslice((cx - 13, cy - 13, cx + 13, cy + 13), 222, 318, fill=C.bg_top)
    d.ellipse((cx - 4, cy + 7, cx + 4, cy + 15), fill=(0, 0, 0))
    d.rounded_rectangle((1980, 52, 2044, 85), radius=9, outline=(110, 110, 110), width=4)
    d.rectangle((2047, 62, 2054, 75), fill=(110, 110, 110))
    d.rounded_rectangle((1986, 58, 2028, 79), radius=5, fill=(89, 220, 123))


def menu(d: ImageDraw.ImageDraw):
    d.ellipse((64, 150, 196, 282), fill=(255, 255, 255))
    for y in (206, 225, 244):
        d.rounded_rectangle((104, y, 162, y + 7), radius=4, fill=C.text)


def bottom_bar(img: Image.Image, active: int):
    d = ImageDraw.Draw(img)
    layer = Image.new("RGBA", img.size, (0, 0, 0, 0))
    ld = ImageDraw.Draw(layer)
    ld.rounded_rectangle((548, 2520, 1516, 2650), radius=70, fill=(34, 50, 76, 26))
    layer = layer.filter(ImageFilter.GaussianBlur(20))
    img.alpha_composite(layer)
    d.rounded_rectangle((548, 2498, 1516, 2634), radius=68, fill=(255, 255, 255), outline=C.line, width=2)
    centers = [(760, 2566), (1032, 2566), (1304, 2566)]
    for i, (cx, cy) in enumerate(centers):
        if i == active:
            d.ellipse((cx - 66, cy - 66, cx + 66, cy + 66), fill=(0, 0, 0))
            color = (255, 255, 255)
        else:
            d.ellipse((cx - 60, cy - 60, cx + 60, cy + 60), fill=(248, 249, 251))
            color = C.text
        if i == 0:
            icon_list_pen(d, cx, cy, color)
        elif i == 1:
            icon_compass(d, cx, cy, color)
        else:
            icon_bell(d, cx, cy - 8, color, 38)
    glow = Image.new("RGBA", img.size, (0, 0, 0, 0))
    gd = ImageDraw.Draw(glow)
    gd.ellipse((1646, 2464, 1930, 2748), fill=(81, 235, 153, 72))
    glow = glow.filter(ImageFilter.GaussianBlur(36))
    img.alpha_composite(glow)
    d.ellipse((1662, 2480, 1914, 2732), fill=(182, 250, 210), outline=(199, 238, 220), width=2)
    d.ellipse((1702, 2520, 1874, 2692), fill=(76, 224, 128))
    icon_mic(d, 1788, 2595, (255, 255, 255), 0.84)
    d.line((820, 2712, 1244, 2712), fill=(0, 0, 0), width=12)


def stat_tile(d: ImageDraw.ImageDraw, xy: tuple[int, int, int, int], icon: str, value: str, label: str, color):
    x1, y1, x2, y2 = xy
    d.rounded_rectangle(xy, radius=32, fill=(255, 255, 255), outline=C.line, width=2)
    cx = (x1 + x2) // 2
    if icon == "bell":
        icon_bell(d, cx, y1 + 74, color, 44)
    elif icon == "check":
        icon_check(d, cx, y1 + 78, color, 44)
    else:
        icon_inbox(d, cx, y1 + 78, color, 46)
    draw_text(d, (cx, y1 + 122), value, 68, C.text, bold=True, anchor="ma")
    draw_text(d, (cx, y1 + 205), label, 34, C.light, bold=True, anchor="ma")


def service_item(img: Image.Image, xy: tuple[int, int, int, int], title: str, body: str, step: str, meta: str, status: str, color):
    d = ImageDraw.Draw(img)
    draw_shadow_card(img, xy, radius=26, shadow_alpha=16)
    x1, y1, x2, y2 = xy
    draw_text(d, (x1 + 42, y1 + 40), ellipsize(d, title, font(42, True), x2 - x1 - 240), 42, C.text, bold=True)
    draw_pill(d, (x2 - 178, y1 + 34, x2 - 42, y1 + 88), status, (248, 250, 252), C.line, color, size=29)
    multiline(d, (x1 + 42, y1 + 115), body, 34, x2 - x1 - 84, 2, C.muted, bold=True, line_gap=13)
    draw_text(d, (x1 + 42, y1 + 222), "下一步：", 34, C.text, bold=True)
    draw_text(d, (x1 + 178, y1 + 222), ellipsize(d, step, font(34, True), x2 - x1 - 250), 34, C.muted, bold=True)
    draw_text(d, (x1 + 42, y2 - 58), ellipsize(d, meta, font(29), x2 - x1 - 84), 29, C.light)


def metric_block(d: ImageDraw.ImageDraw, x: int, y: int, value: str, label: str, color=C.green_dark):
    d.rounded_rectangle((x, y, x + 220, y + 128), radius=35, fill=C.green_soft)
    draw_text(d, (x + 110, y + 20), value, 48, color, bold=True, anchor="ma")
    draw_text(d, (x + 110, y + 82), label, 27, color, bold=True, anchor="ma")


def screen_service() -> Image.Image:
    img = canvas()
    d = ImageDraw.Draw(img)
    status_bar(d)
    menu(d)
    draw_text(d, (96, 360), "服务提醒", 72, C.text, bold=True)
    multiline(d, (96, 462), "AI 自动梳理今日要服务谁、为什么提醒和下一步动作，帮助园所把跟进做扎实。", 43, 1220, 2, C.muted, bold=True)
    d.rounded_rectangle((1608, 382, 1760, 452), radius=35, fill=(255, 255, 255), outline=C.line, width=2)
    icon_doc(d, 1659, 389, C.muted)
    icon_refresh(d, 1880, 414, C.muted, 56)

    stat_tile(d, (96, 645, 610, 915), "bell", "12", "待处理", C.orange)
    stat_tile(d, (776, 645, 1290, 915), "check", "38", "已完成", C.green)
    stat_tile(d, (1454, 645, 1968, 915), "inbox", "50", "全部", C.muted)

    draw_text(d, (96, 1018), "高价值提醒", 52, C.text, bold=True)
    draw_text(d, (1218, 1018), "服务看板", 52, C.text, bold=True)
    service_item(img, (96, 1100, 1110, 1435), "开放日家长邀约与接待确认", "本周六开放日已预约 32 位家长，3 组需确认到场时间和试听需求。", "优先联系未确认家庭，发送到园路线与课程安排", "招生增长 · 高意向 · 今日 10:30 · 负责人：张老师", "待处理", C.orange)
    service_item(img, (96, 1490, 1110, 1825), "试听课后黄金 24 小时跟进", "小明妈妈昨天完成试听课，系统识别为高意向客户，关注课程效果与接送便利。", "发送个性化反馈，并预约下次沟通时间", "客户跟进 · 高优先级 · 已生成 2 条话术", "高优先", C.green_dark)

    draw_shadow_card(img, (1218, 1100, 1968, 1538), radius=30, fill=(252, 255, 253), outline=(218, 242, 235), shadow_alpha=16)
    draw_text(d, (1266, 1152), "本周服务进展", 45, C.text, bold=True)
    metric_block(d, 1266, 1242, "86%", "响应率")
    metric_block(d, 1538, 1242, "12", "新增线索")
    metric_block(d, 1266, 1410, "7", "已预约试听")
    metric_block(d, 1538, 1410, "24h", "跟进时效")

    draw_shadow_card(img, (1218, 1598, 1968, 2070), radius=30, shadow_alpha=16)
    draw_text(d, (1266, 1650), "今日优先动作", 45, C.text, bold=True)
    actions = [
        ("10:30", "确认 3 组开放日到场时间"),
        ("14:00", "发送试听课个性化反馈"),
        ("17:30", "复盘本周招生转化线索"),
    ]
    for i, (time, text) in enumerate(actions):
        y = 1742 + i * 104
        d.rounded_rectangle((1266, y, 1378, y + 54), radius=27, fill=C.green_soft)
        draw_text(d, (1322, y + 10), time, 25, C.green_dark, bold=True, anchor="ma")
        draw_text(d, (1410, y + 4), text, 33, C.muted, bold=True)
    bottom_bar(img, 2)
    return img.convert("RGB")


def knowledge_tile(img: Image.Image, xy, title: str, desc: str, count: str, accent, soft):
    d = ImageDraw.Draw(img)
    draw_shadow_card(img, xy, radius=30, shadow_alpha=16)
    x1, y1, x2, y2 = xy
    d.rounded_rectangle((x1 + 44, y1 + 44, x1 + 140, y1 + 140), radius=26, fill=soft)
    icon_doc(d, x1 + 68, y1 + 62, accent)
    draw_text(d, (x1 + 172, y1 + 54), title, 45, C.text, bold=True)
    multiline(d, (x1 + 44, y1 + 172), desc, 34, x2 - x1 - 88, 2, C.muted, bold=True)
    draw_text(d, (x1 + 44, y2 - 78), count, 39, accent, bold=True)


def memory_row(img: Image.Image, xy, title: str, body: str, date: str, badge: str, action: str):
    d = ImageDraw.Draw(img)
    draw_shadow_card(img, xy, radius=30, shadow_alpha=16)
    x1, y1, x2, y2 = xy
    draw_text(d, (x1 + 52, y1 + 50), ellipsize(d, title, font(44, True), x2 - x1 - 100), 44, C.text, bold=True)
    multiline(d, (x1 + 52, y1 + 125), body, 36, x2 - x1 - 104, 2, C.muted, bold=True, line_gap=14)
    draw_text(d, (x1 + 52, y2 - 80), date, 34, C.light, bold=True)
    draw_pill(d, (x2 - 612, y2 - 98, x2 - 396, y2 - 38), badge, C.green_soft, (179, 233, 215), C.green_dark, size=30)
    draw_pill(d, (x2 - 356, y2 - 98, x2 - 84, y2 - 38), action, (226, 247, 244), (226, 247, 244), C.green, size=30)


def screen_knowledge() -> Image.Image:
    img = canvas()
    d = ImageDraw.Draw(img)
    status_bar(d)
    menu(d)
    draw_text(d, (96, 365), "知识库", 72, C.text, bold=True)
    d.rounded_rectangle((1734, 352, 1968, 466), radius=57, fill=(255, 255, 255), outline=C.line, width=2)
    draw_text(d, (1786, 388), "更多", 39, C.muted, bold=True)
    d.line((1900, 398, 1919, 417, 1900, 436), fill=C.muted, width=5)

    knowledge_tile(img, (96, 590, 948, 922), "招生话术", "试听邀约、异议处理、续费跟进", "128 个内容", C.green_dark, C.green_soft)
    knowledge_tile(img, (1044, 590, 1968, 922), "运营支持", "园所 SOP、活动清单、教师培训", "64 个流程", C.blue, C.blue_soft)
    knowledge_tile(img, (96, 982, 948, 1314), "家长服务", "沟通记录、满意度回访、问题追踪", "96 条经验", C.orange, C.orange_soft)
    knowledge_tile(img, (1044, 982, 1968, 1314), "活动策划", "开放日、节日活动、招生转化方案", "42 个模板", C.green_dark, C.green_soft)

    draw_text(d, (96, 1452), "全部记忆", 61, C.text, bold=True)
    d.line((344, 1480, 371, 1507, 398, 1480), fill=C.text, width=6)
    d.rounded_rectangle((166, 1550, 257, 1564), radius=7, fill=C.text)
    memory_row(img, (96, 1630, 1968, 1965), "开放日家长沟通纪要", "AI 已整理 18 条家长问题，提取出价格、接送、课程效果三类关注点，并生成跟进建议。", "9月5日 11:15", "转写成功", "提取 6 条服务")
    memory_row(img, (96, 2030, 1968, 2365), "试听课后跟进录音", "录音 12 分钟已完成转写，自动总结孩子课堂表现、家长疑虑和下次邀约时机。", "9月4日 16:28", "已生成摘要", "提取 4 条服务")
    bottom_bar(img, 0)
    return img.convert("RGB")


def topic_card(img: Image.Image, xy, title: str, body: str, meta: str, tag: str, accent, soft):
    d = ImageDraw.Draw(img)
    draw_shadow_card(img, xy, radius=28, shadow_alpha=16)
    x1, y1, x2, y2 = xy
    d.rounded_rectangle((x1 + 42, y1 + 42, x1 + 202, y1 + 202), radius=24, fill=soft)
    icon_doc(d, x1 + 98, y1 + 78, accent)
    draw_text(d, (x1 + 122, y1 + 146), tag, 30, accent, bold=True, anchor="ma")
    draw_text(d, (x1 + 242, y1 + 45), ellipsize(d, title, font(41, True), x2 - x1 - 300), 41, C.text, bold=True)
    multiline(d, (x1 + 242, y1 + 108), body, 34, x2 - x1 - 300, 2, C.muted, bold=True)
    draw_text(d, (x1 + 242, y2 - 72), meta, 31, C.light, bold=True)


def chip(d: ImageDraw.ImageDraw, x: int, y: int, w: int, text: str, active=False):
    if active:
        d.rounded_rectangle((x, y, x + w, y + 82), radius=41, fill=(235, 253, 248), outline=(174, 238, 219), width=3)
        draw_text(d, (x + w // 2, y + 20), text, 34, C.green_dark, bold=True, anchor="ma")
    else:
        d.rounded_rectangle((x, y, x + w, y + 82), radius=41, fill=(248, 250, 252), outline=C.line, width=2)
        draw_text(d, (x + w // 2, y + 20), text, 34, C.muted, bold=True, anchor="ma")


def screen_topics() -> Image.Image:
    img = canvas()
    d = ImageDraw.Draw(img)
    status_bar(d)
    menu(d)
    draw_text(d, (96, 365), "精选主题", 72, C.text, bold=True)
    icon_refresh(d, 1768, 405, C.green, 42)
    draw_text(d, (1826, 378), "换一批", 39, C.green, bold=True)
    topic_card(img, (96, 560, 1968, 842), "家长犹豫不报名？4 类顾虑逐个击破", "识别价格、信任、接送、家庭决策四类顾虑，自动生成针对性跟进策略。", "9月5日 11:21  @创作者", "招生", C.green_dark, (213, 238, 226))
    topic_card(img, (96, 912, 1968, 1194), "开放日高转化：从展示到成交的 5 步 SOP", "优化入园接待、儿童体验、透明答疑和会后追踪，降低家长决策阻力。", "9月5日 11:20  @创作者", "运营", C.blue, C.blue_soft)
    chip(d, 96, 1300, 198, "推荐 18", True)
    chip(d, 330, 1300, 286, "招生增长 12")
    chip(d, 652, 1300, 260, "家长服务 9")
    chip(d, 948, 1300, 260, "活动策划 6")
    draw_text(d, (96, 1468), "推荐", 58, C.text, bold=True)
    draw_text(d, (1906, 1476), "18 条", 39, C.light, bold=True, anchor="ra")
    topic_card(img, (96, 1585, 1004, 1878), "老生续费提醒：提前 30 天建立信任", "按孩子成长记录、课堂反馈和家长关注点，拆解续费沟通节奏。", "6 个步骤 · 3 条话术", "续费", C.orange, C.orange_soft)
    topic_card(img, (1060, 1585, 1968, 1878), "朋友圈招生文案：一周 7 条可直接发", "从真实课堂、孩子变化、家长反馈中提炼内容，持续触达潜在客户。", "7 条模板 · 今日更新", "文案", C.green_dark, C.green_soft)
    topic_card(img, (96, 1945, 1004, 2238), "园所活动复盘：把经验沉淀成 SOP", "活动数据、家长反馈和团队协作自动整理，形成下一次可复用清单。", "4 个模块 · 可分享", "复盘", C.blue, C.blue_soft)
    topic_card(img, (1060, 1945, 1968, 2238), "家长满意度回访：问题提前发现", "定期提取服务风险点，让班主任及时跟进，减少流失。", "5 条模板 · 高优先", "服务", C.green_dark, (213, 238, 226))
    bottom_bar(img, 1)
    return img.convert("RGB")


def main() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    outputs = [
        ("01-service-reminders-ipad-13.png", screen_service()),
        ("02-knowledge-base-ipad-13.png", screen_knowledge()),
        ("03-featured-topics-ipad-13.png", screen_topics()),
    ]
    for name, img in outputs:
        path = OUT_DIR / name
        img.save(path, "PNG", optimize=True)
        print(path)


if __name__ == "__main__":
    main()
