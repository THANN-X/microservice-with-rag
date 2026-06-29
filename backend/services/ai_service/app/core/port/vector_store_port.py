"""Port (interface) for vector store operations."""

from abc import ABC, abstractmethod

from app.core.domain.models import ProductDocument, ProductResult


class VectorStorePort(ABC):
    @abstractmethod
    async def upsert_product(self, product: ProductDocument, embedding: list[float]) -> None:
        """Insert or update a product in the vector store."""
        ...

    @abstractmethod
    async def delete_product(self, product_id: int) -> None:
        """Delete a product from the vector store."""
        ...

    @abstractmethod
    async def search(self, query_embedding: list[float], top_k: int = 5, score_threshold: float = 0.5) -> list[ProductResult]:
        """Search for similar products by embedding vector."""
        ...

    @abstractmethod
    async def ensure_collection(self) -> None:
        """Create the collection if it doesn't exist."""
        ...

    @abstractmethod
    async def get_product(self, product_id: int) -> ProductDocument | None:
        """Retrieve a product's full payload from the vector store."""
        ...
