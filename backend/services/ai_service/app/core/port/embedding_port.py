"""Port (interface) for embedding operations."""

from abc import ABC, abstractmethod


class EmbeddingPort(ABC):
    @abstractmethod
    def embed(self, text: str) -> list[float]:
        """Generate embedding vector from text."""
        ...

    @abstractmethod
    def embed_batch(self, texts: list[str]) -> list[list[float]]:
        """Generate embedding vectors for a batch of texts."""
        ...
