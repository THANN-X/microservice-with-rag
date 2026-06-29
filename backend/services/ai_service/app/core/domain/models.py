"""Domain models for AI chat service."""

from dataclasses import dataclass, field
from typing import Optional


@dataclass
class ProductDocument:
    """Product data to be embedded and stored in vector DB."""

    product_id: int
    name: str
    description: str
    categories: list[str] = field(default_factory=list)
    variants: list["VariantInfo"] = field(default_factory=list)
    is_active: bool = True
    is_deleted: bool = False

    def to_embedding_text(self) -> str:
        """Combine product info into a single text for embedding."""
        parts = [self.name, self.description]
        if self.categories:
            parts.append("หมวดหมู่: " + ", ".join(self.categories))
        for v in self.variants:
            parts.append(f"{v.name} ราคา {v.price} บาท")
            if v.attributes:
                parts.append(" ".join(f"{k}: {val}" for k, val in v.attributes.items()))
        return " | ".join(parts)

    def min_price(self) -> float:
        if not self.variants:
            return 0.0
        prices = [v.price for v in self.variants if v.price > 0]
        return min(prices) if prices else 0.0


@dataclass
class VariantInfo:
    variant_id: int
    sku: str
    name: str
    price: float
    stock: int
    is_active: bool = True
    attributes: dict[str, str] = field(default_factory=dict)


@dataclass
class ChatMessage:
    role: str  # "user" | "assistant"
    content: str


@dataclass
class ChatRequest:
    message: str
    session_id: str


@dataclass
class ChatResponseChunk:
    event_type: str
    text_content: str = ""
    product_ids: list[int] = field(default_factory=list)


@dataclass
class ProductResult:
    product_id: int
    name: str
    description: str
    min_price: float
    relevance_score: float
    variants: list[dict] = field(default_factory=list)
