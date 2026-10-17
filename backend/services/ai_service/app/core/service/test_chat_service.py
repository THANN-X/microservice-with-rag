"""Smoke test เริ่มต้น — ทดสอบ _build_context ที่เป็น pure function"""

from app.core.service.chat_service import ChatService


def test_build_context_empty():
    # ไม่มีสินค้า ต้องได้ข้อความว่าไม่พบ
    result = ChatService._build_context([])
    assert "ไม่พบสินค้า" in result
