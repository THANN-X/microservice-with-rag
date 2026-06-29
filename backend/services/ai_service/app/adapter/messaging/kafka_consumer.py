"""Kafka consumer adapter — consumes product events and indexes into vector store."""

import asyncio
import json
import logging
import threading

from confluent_kafka import Consumer, KafkaError

from app.config import settings
from app.core.domain.models import ProductDocument, VariantInfo
from app.core.port.embedding_port import EmbeddingPort
from app.core.port.vector_store_port import VectorStorePort

logger = logging.getLogger(__name__)


class ProductEventConsumer:
    """Consumes product.events from Kafka and keeps the vector store in sync."""

    def __init__(
        self,
        vector_store: VectorStorePort,
        embedding: EmbeddingPort,
    ):
        self._vector_store = vector_store
        self._embedding = embedding
        self._running = False
        self._thread: threading.Thread | None = None
        self._loop: asyncio.AbstractEventLoop | None = None

    def start(self, loop: asyncio.AbstractEventLoop) -> None:
        self._loop = loop
        self._running = True
        self._thread = threading.Thread(target=self._consume_loop, daemon=True)
        self._thread.start()
        logger.info("Kafka consumer started for topic: %s", settings.KAFKA_TOPIC)

    def stop(self) -> None:
        self._running = False
        if self._thread:
            self._thread.join(timeout=10)

    def _consume_loop(self) -> None:
        consumer = Consumer(
            {
                "bootstrap.servers": settings.KAFKA_BROKERS,
                "group.id": settings.KAFKA_GROUP_ID,
                "auto.offset.reset": "earliest",
                "enable.auto.commit": False,
            }
        )
        consumer.subscribe([settings.KAFKA_TOPIC])

        try:
            while self._running:
                msg = consumer.poll(timeout=1.0)
                if msg is None:
                    continue
                if msg.error():
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        continue
                    logger.error("Kafka error: %s", msg.error())
                    continue

                try:
                    self._handle_message(msg)
                    consumer.commit(msg, asynchronous=False)
                except Exception:
                    logger.exception("Error handling Kafka message")
        finally:
            consumer.close()

    def _handle_message(self, msg) -> None:
        # Read event type from header or body
        event_type = self._extract_event_type(msg)
        if not event_type:
            return

        body = json.loads(msg.value().decode("utf-8"))

        handler_map = {
            "PRODUCT_CREATED": self._handle_product_created,
            "PRODUCT_INFO_UPDATED": self._handle_product_info_updated,
            "PRODUCT_VARIANT_ADDED": self._handle_variant_added,
            "PRODUCT_PRICE_CHANGED": self._handle_price_changed,
            "PRODUCT_DELETED": self._handle_product_deleted,
            "STOCK_ADJUSTED": self._handle_stock_adjusted,
            "STOCK_UPDATED": self._handle_stock_adjusted,
            "PRODUCT_ACTIVE_CHANGED": self._handle_product_active_changed,
            "PRODUCT_VARIANT_ACTIVE_CHANGED": self._handle_variant_active_changed,
            "PRODUCT_CATEGORIES_UPDATED": self._handle_product_categories_updated,
        }

        handler = handler_map.get(event_type)
        if handler:
            logger.info("Processing event: %s", event_type)
            handler(body)
        else:
            logger.debug("Ignoring event type: %s", event_type)

    def _extract_event_type(self, msg) -> str | None:
        # Try Kafka header first
        headers = msg.headers() or []
        for key, value in headers:
            if key == "EventType" and value:
                return value.decode("utf-8")

        # Fallback: peek JSON body
        try:
            body = json.loads(msg.value().decode("utf-8"))
            return body.get("event_type")
        except (json.JSONDecodeError, UnicodeDecodeError):
            return None

    def _get_product_sync(self, product_id: int) -> ProductDocument | None:
        if not self._loop:
            return None
        future = asyncio.run_coroutine_threadsafe(
            self._vector_store.get_product(product_id), self._loop
        )
        try:
            return future.result(timeout=5.0)
        except Exception:
            logger.exception("Timeout or error getting product %s", product_id)
            return None

    def _handle_product_created(self, body: dict) -> None:
        self._handle_product_upsert(body)

    def _handle_product_info_updated(self, body: dict) -> None:
        self._handle_product_upsert(body)

    def _handle_product_upsert(self, body: dict) -> None:
        product_id = body.get("product_id") or body.get("ProductID")
        if not product_id:
            logger.warning("Product upsert event payload missing product ID")
            return
            
        name = body.get("name") or body.get("Name") or ""
        description = body.get("description") or body.get("Description") or ""

        product = self._get_product_sync(product_id)
        if product:
            product.name = name or product.name
            product.description = description or product.description
        else:
            product = ProductDocument(
                product_id=product_id,
                name=name,
                description=description,
            )
        self._embed_and_upsert(product)

    def _handle_variant_added(self, body: dict) -> None:
        product_id = body.get("product_id") or body.get("ProductID")
        variant_id = body.get("variant_id") or body.get("VariantID")
        if not product_id or not variant_id:
            logger.warning("PRODUCT_VARIANT_ADDED payload missing product_id or variant_id")
            return
        
        product = self._get_product_sync(product_id)
        if not product:
            logger.warning("Product %s not found in vector store to add variant", product_id)
            return

        attrs = {}
        for attr in body.get("attributes", []):
            if isinstance(attr, dict) and "key" in attr:
                attrs[attr["key"]] = attr.get("value", "")

        variant = VariantInfo(
            variant_id=variant_id,
            sku=body.get("sku") or body.get("Sku") or "",
            name=body.get("name") or body.get("Name") or "",
            price=float(body.get("price") or body.get("Price") or 0.0),
            stock=int(body.get("stock") or body.get("Stock") or 0),
            attributes=attrs,
        )
        
        # Add or update variant
        existing_idx = next((i for i, v in enumerate(product.variants) if v.variant_id == variant.variant_id), -1)
        if existing_idx >= 0:
            product.variants[existing_idx] = variant
        else:
            product.variants.append(variant)

        self._embed_and_upsert(product)

    def _handle_price_changed(self, body: dict) -> None:
        product_id = body.get("product_id") or body.get("ProductID")
        variant_id = body.get("variant_id") or body.get("VariantID")
        new_price = body.get("new_price") or body.get("NewPrice") or 0.0
        if not product_id or not variant_id:
            logger.warning("PRODUCT_PRICE_CHANGED payload missing product_id or variant_id")
            return
            
        logger.info(
            "Price changed for product %s variant %s: %s -> %s",
            product_id,
            variant_id,
            body.get("old_price") or body.get("OldPrice"),
            new_price,
        )
        
        product = self._get_product_sync(product_id)
        if not product:
            return
            
        updated = False
        for v in product.variants:
            if v.variant_id == variant_id:
                v.price = float(new_price)
                updated = True
                break
                
        if updated:
            self._embed_and_upsert(product)

    def _handle_product_deleted(self, body: dict) -> None:
        product_id = body.get("product_id") or body.get("ProductID")
        if not product_id:
            return
        if self._loop:
            future = asyncio.run_coroutine_threadsafe(
                self._vector_store.delete_product(product_id), self._loop
            )
            try:
                future.result(timeout=10.0)
            except Exception:
                logger.exception("Failed to delete product %s", product_id)
                raise

    def _handle_stock_adjusted(self, body: dict) -> None:
        product_id = body.get("product_id") or body.get("ProductID")
        variant_id = body.get("variant_id") or body.get("VariantID")
        new_stock = body.get("new_stock") or body.get("NewStock")
        if not product_id or not variant_id or new_stock is None:
            logger.warning("STOCK_ADJUSTED/UPDATED payload missing product_id, variant_id or new_stock")
            return
            
        logger.info(
            "Stock adjusted/updated for product %s variant %s: %s -> %s",
            product_id,
            variant_id,
            body.get("old_stock") or body.get("OldStock"),
            new_stock,
        )
        
        product = self._get_product_sync(product_id)
        if not product:
            return
            
        updated = False
        for v in product.variants:
            if v.variant_id == variant_id:
                v.stock = int(new_stock)
                updated = True
                break
                
        if updated:
            self._embed_and_upsert(product)

    def _handle_product_active_changed(self, body: dict) -> None:
        product_id = body.get("product_id") or body.get("ProductID")
        is_active = body.get("is_active")
        if is_active is None:
            is_active = body.get("IsActive", True)
            
        if not product_id:
            return
            
        product = self._get_product_sync(product_id)
        if not product:
            return
            
        product.is_active = bool(is_active)
        self._embed_and_upsert(product)

    def _handle_variant_active_changed(self, body: dict) -> None:
        product_id = body.get("product_id") or body.get("ProductID")
        variant_id = body.get("variant_id") or body.get("VariantID")
        is_active = body.get("is_active")
        if is_active is None:
            is_active = body.get("IsActive", True)
            
        if not product_id or not variant_id:
            return
            
        product = self._get_product_sync(product_id)
        if not product:
            return
            
        updated = False
        for v in product.variants:
            if v.variant_id == variant_id:
                v.is_active = bool(is_active)
                updated = True
                break
                
        if updated:
            self._embed_and_upsert(product)

    def _handle_product_categories_updated(self, body: dict) -> None:
        product_id = body.get("product_id") or body.get("ProductID")
        categories_data = body.get("categories") or body.get("Categories", [])
        if not product_id:
            return
            
        product = self._get_product_sync(product_id)
        if not product:
            return
            
        categories = []
        for cat in categories_data:
            if isinstance(cat, dict):
                name = cat.get("name") or cat.get("Name")
                if name:
                    categories.append(name)
                    
        product.categories = categories
        self._embed_and_upsert(product)

    def _embed_and_upsert(self, product: ProductDocument) -> None:
        text = product.to_embedding_text()
        embedding = self._embedding.embed(text)
        if self._loop:
            future = asyncio.run_coroutine_threadsafe(
                self._vector_store.upsert_product(product, embedding), self._loop
            )
            try:
                future.result(timeout=10.0)
            except Exception:
                logger.exception("Failed to upsert product %s", product.product_id)
                raise
