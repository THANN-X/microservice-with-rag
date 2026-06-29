"""AI Service — Entry point (Composition Root).

Wires all adapters together and starts:
1. gRPC server (for BFF queries)
2. Kafka consumer (for product event sync)
3. FastAPI HTTP server (for health checks)
"""

import asyncio
import logging
import signal

import uvicorn
from fastapi import FastAPI
from qdrant_client import AsyncQdrantClient

from app.adapter.embedding.sentence_transformer import SentenceTransformerEmbedding
from app.adapter.grpc.chat_grpc_server import serve_grpc
from app.adapter.messaging.kafka_consumer import ProductEventConsumer
from app.adapter.repository.qdrant_store import QdrantVectorStore
from app.config import settings
from app.core.service.chat_service import ChatService

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)

# --- FastAPI (health check only) ---
http_app = FastAPI(title="AI Service", docs_url=None, redoc_url=None)


@http_app.get("/health")
async def health():
    return {"status": "ok"}


async def main():
    logger.info("AI Service starting...")

    # --- Adapters ---
    qdrant_client = AsyncQdrantClient(host=settings.QDRANT_HOST, port=settings.QDRANT_PORT)
    vector_store = QdrantVectorStore(qdrant_client)
    await vector_store.ensure_collection()

    embedding = SentenceTransformerEmbedding()

    # --- Core Service (LangChain RAG chain) ---
    chat_service = ChatService(
        vector_store=vector_store,
        embedding=embedding,
    )

    # --- gRPC Server ---
    grpc_server = await serve_grpc(chat_service, settings.GRPC_PORT)

    # --- Kafka Consumer ---
    loop = asyncio.get_running_loop()
    kafka_consumer = ProductEventConsumer(
        vector_store=vector_store,
        embedding=embedding,
    )
    kafka_consumer.start(loop)

    # --- HTTP Server (health check) ---
    config = uvicorn.Config(
        http_app,
        host="0.0.0.0",
        port=settings.HTTP_PORT,
        log_level="info",
    )
    http_server = uvicorn.Server(config)

    # --- Graceful shutdown ---
    stop_event = asyncio.Event()

    def _signal_handler():
        logger.info("Shutdown signal received")
        stop_event.set()

    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, _signal_handler)

    # Run HTTP server in background
    http_task = asyncio.create_task(http_server.serve())

    logger.info(
        "AI Service ready — gRPC:%d, HTTP:%d",
        settings.GRPC_PORT,
        settings.HTTP_PORT,
    )

    await stop_event.wait()

    # Cleanup
    logger.info("Shutting down...")
    kafka_consumer.stop()
    await grpc_server.stop(grace=5)
    http_server.should_exit = True
    await http_task
    await qdrant_client.close()
    logger.info("AI Service stopped.")


if __name__ == "__main__":
    asyncio.run(main())
