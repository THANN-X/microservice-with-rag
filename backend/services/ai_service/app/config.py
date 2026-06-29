from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # gRPC
    GRPC_PORT: int = 50051

    # HTTP (health check)
    HTTP_PORT: int = 8000

    # Qdrant
    QDRANT_HOST: str = "localhost"
    QDRANT_PORT: int = 6333
    QDRANT_COLLECTION: str = "products"

    # Kafka
    KAFKA_BROKERS: str = "localhost:9094"
    KAFKA_TOPIC: str = "product.events"
    KAFKA_GROUP_ID: str = "ai-service-group-v2"

    # Embedding
    EMBEDDING_MODEL: str = "sentence-transformers/paraphrase-multilingual-mpnet-base-v2"
    EMBEDDING_DIM: int = 768

    # LLM (Gemini)
    GOOGLE_API_KEY: str = ""
    GEMINI_MODEL: str = "gemini-3.1-flash-lite"

    # Conversation memory
    CONVERSATION_MAX_HISTORY: int = 10

    class Config:
        env_file = ".env"


settings = Settings()
