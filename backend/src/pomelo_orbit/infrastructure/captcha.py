"""验证码生成和验证"""

from __future__ import annotations

import base64
import io
import random
from datetime import UTC, datetime, timedelta
from typing import TYPE_CHECKING

from jose import jwt
from PIL import Image, ImageDraw, ImageFont

if TYPE_CHECKING:
    from PIL.ImageFont import FreeTypeFont
    from PIL.ImageFont import ImageFont as ImageFontType


def generate_captcha_text(length: int = 4) -> str:
    """生成验证码文本（数字+大写字母，排除易混淆字符）"""
    # 排除 0O1Il 等易混淆字符
    chars = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
    return "".join(random.choice(chars) for _ in range(length))


def create_captcha_image(text: str, width: int = 120, height: int = 40) -> str:
    """
    创建验证码图片并返回 base64 编码

    Args:
        text: 验证码文本
        width: 图片宽度
        height: 图片高度

    Returns:
        base64 编码的图片字符串
    """
    # 创建图片
    image = Image.new("RGB", (width, height), color=(255, 255, 255))
    draw = ImageDraw.Draw(image)

    # 使用默认字体
    font: FreeTypeFont | ImageFontType
    try:
        font = ImageFont.truetype("arial.ttf", 28)
    except OSError:
        font = ImageFont.load_default()

    # 绘制文本
    text_bbox = draw.textbbox((0, 0), text, font=font)
    text_width = text_bbox[2] - text_bbox[0]
    text_height = text_bbox[3] - text_bbox[1]
    x = (width - text_width) // 2
    y = (height - text_height) // 2

    # 添加随机颜色
    color = (random.randint(0, 100), random.randint(0, 100), random.randint(0, 100))
    draw.text((x, y), text, fill=color, font=font)

    # 添加干扰线
    for _ in range(3):
        x1 = random.randint(0, width)
        y1 = random.randint(0, height)
        x2 = random.randint(0, width)
        y2 = random.randint(0, height)
        draw.line([(x1, y1), (x2, y2)], fill=(200, 200, 200), width=1)

    # 添加噪点
    for _ in range(50):
        x = random.randint(0, width - 1)
        y = random.randint(0, height - 1)
        draw.point((x, y), fill=(150, 150, 150))

    # 转换为 base64
    buffer = io.BytesIO()
    image.save(buffer, format="PNG")
    image_base64 = base64.b64encode(buffer.getvalue()).decode()

    return f"data:image/png;base64,{image_base64}"


def generate_captcha_token(answer: str, secret_key: str, ttl_minutes: int = 1) -> str:
    """
    生成验证码 token（JWT）

    Args:
        answer: 验证码答案
        secret_key: JWT 密钥
        ttl_minutes: 有效期（分钟）

    Returns:
        JWT token
    """
    exp = datetime.now(UTC) + timedelta(minutes=ttl_minutes)
    payload = {"answer": answer, "exp": exp, "type": "captcha"}
    token: str = jwt.encode(payload, secret_key, algorithm="HS256")
    return token


def verify_captcha_token(token: str, user_answer: str, secret_key: str) -> bool:
    """
    验证验证码 token

    Args:
        token: JWT token
        user_answer: 用户输入的答案
        secret_key: JWT 密钥

    Returns:
        验证是否通过
    """
    try:
        payload = jwt.decode(token, secret_key, algorithms=["HS256"])
        if payload.get("type") != "captcha":
            return False
        answer: str = payload.get("answer", "")
        result: bool = answer.upper() == user_answer.upper()
        return result
    except jwt.JWTError:
        return False


__all__ = [
    "create_captcha_image",
    "generate_captcha_text",
    "generate_captcha_token",
    "verify_captcha_token",
]
