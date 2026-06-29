"""Qdrant vector store adapter — implements VectorStorePort."""

import logging

from qdrant_client import AsyncQdrantClient
from qdrant_client.models import (
    Distance,
    FieldCondition,
    Filter,
    MatchValue,
    PointStruct,
    VectorParams,
)

from app.config import settings
from app.core.domain.models import ProductDocument, ProductResult, VariantInfo
from app.core.port.vector_store_port import VectorStorePort

logger = logging.getLogger(__name__)


class QdrantVectorStore(VectorStorePort):
    def __init__(self, client: AsyncQdrantClient):
        self._client = client
        self._collection = settings.QDRANT_COLLECTION

    async def ensure_collection(self) -> None:
        collections = await self._client.get_collections()
        names = [c.name for c in collections.collections]
        
        if self._collection in names:
            try:
                info = await self._client.get_collection(self._collection)
                if hasattr(info.config.params.vectors, 'size'):
                    current_dim = info.config.params.vectors.size
                    if current_dim != settings.EMBEDDING_DIM:
                        logger.warning("Dimension mismatch for collection %s: expected %d, got %d. Deleting collection...", self._collection, settings.EMBEDDING_DIM, current_dim)
                        await self._client.delete_collection(self._collection)
                        names.remove(self._collection)
            except Exception as e:
                logger.error("Error checking collection dimension: %s", e)

        if self._collection not in names:
            await self._client.create_collection(
                collection_name=self._collection,
                vectors_config=VectorParams(
                    size=settings.EMBEDDING_DIM,
                    distance=Distance.COSINE,
                ),
            )
            logger.info("Created Qdrant collection: %s", self._collection)

    async def upsert_product(self, product: ProductDocument, embedding: list[float]) -> None:
        payload = {
            "product_id": product.product_id,
            "name": product.name,
            "description": product.description,
            "categories": product.categories,
            "min_price": product.min_price(),
            "variants": [
                {
                    "variant_id": v.variant_id,
                    "name": v.name,
                    "price": v.price,
                    "stock": v.stock,
                    "attributes": v.attributes,
                }
                for v in product.variants
            ],
            "is_active": product.is_active,
        }
        point = PointStruct(
            id=product.product_id,
            vector=embedding,
            payload=payload,
        )
        await self._client.upsert(
            collection_name=self._collection,
            points=[point],
        )

    async def delete_product(self, product_id: int) -> None:
        await self._client.delete(
            collection_name=self._collection,
            points_selector=[product_id],
        )

    async def search(self, query_embedding: list[float], top_k: int = 5, score_threshold: float = 0.5) -> list[ProductResult]:
        results = await self._client.query_points(
            collection_name=self._collection,
            query=query_embedding,
            query_filter=Filter(
                must=[FieldCondition(key="is_active", match=MatchValue(value=True))]
            ),
            limit=top_k,
            score_threshold=score_threshold,
            with_payload=True,
        )

        products: list[ProductResult] = []
        for point in results.points:
            payload = point.payload or {}
            products.append(
                ProductResult(
                    product_id=payload.get("product_id", 0),
                    name=payload.get("name", ""),
                    description=payload.get("description", ""),
                    min_price=payload.get("min_price", 0.0),
                    relevance_score=point.score,
                    variants=payload.get("variants", []),
                )
            )
        return products

    async def get_product(self, product_id: int) -> ProductDocument | None:
        try:
            results = await self._client.retrieve(
                collection_name=self._collection,
                ids=[product_id],
                with_payload=True,
                with_vectors=False
            )
            if not results:
                return None
            
            payload = results[0].payload or {}
            variants = []
            for v in payload.get("variants", []):
                variants.append(VariantInfo(
                    variant_id=v.get("variant_id", 0),
                    sku=v.get("sku", ""),
                    name=v.get("name", ""),
                    price=v.get("price", 0.0),
                    stock=v.get("stock", 0),
                    attributes=v.get("attributes", {}),
                ))
            
            return ProductDocument(
                product_id=payload.get("product_id", product_id),
                name=payload.get("name", ""),
                description=payload.get("description", ""),
                categories=payload.get("categories", []),
                variants=variants,
                is_active=payload.get("is_active", True)
            )
        except Exception as e:
            logger.exception("Error getting product %d from Qdrant", product_id)
            return None
