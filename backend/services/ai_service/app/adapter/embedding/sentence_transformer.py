"""Sentence-Transformers embedding adapter — implements EmbeddingPort."""

from sentence_transformers import SentenceTransformer

from app.config import settings
from app.core.port.embedding_port import EmbeddingPort


class SentenceTransformerEmbedding(EmbeddingPort):
    def __init__(self) -> None:
        self._model = SentenceTransformer(settings.EMBEDDING_MODEL)

    def embed(self, text: str) -> list[float]:
        return self._model.encode(text, normalize_embeddings=True).tolist()

    def embed_batch(self, texts: list[str]) -> list[list[float]]:
        embeddings = self._model.encode(texts, normalize_embeddings=True)
        return [e.tolist() for e in embeddings]
