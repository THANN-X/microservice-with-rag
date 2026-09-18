from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # gRPC
    GRPC_PORT: int = 50051

    # HTTP (health check)
    HTTP_PORT: int = 8000

    # Qdrant
    QDRANT_HOST: str = "localhost"
    QDRANT_PORT: int = 6333
    # ตั้งชื่อ collection ตามโมเดลที่ index ไว้ — เวกเตอร์ที่ index ด้วยคนละโมเดลค้นข้ามกันไม่ได้
    # collection 'products' (mpnet, 768 มิติ) ยังอยู่ ใช้ rollback ได้
    QDRANT_COLLECTION: str = "products_bge_m3"

    # Kafka
    KAFKA_BROKERS: str = "localhost:9094"
    KAFKA_TOPIC: str = "product.events"
    KAFKA_GROUP_ID: str = "ai-service-group-v2"

    # Embedding
    # BGE-M3 เป็นโมเดลสาย retrieval (asymmetric query↔document) ต่างจาก paraphrase-mpnet เดิม
    # ที่เป็น symmetric paraphrase — วัดกับ golden set 34 ข้อได้ hit rate 0.93→1.00
    # และ reject rate (คำถามนอกแคตตาล็อกต้องไม่คืนสินค้า) 0.83→1.00
    # กินแรมราว 2.3GB ตอนโหลด (mpnet ~1GB) ดู mem_limit ของ ai-service-app ใน docker-compose
    EMBEDDING_MODEL: str = "BAAI/bge-m3"
    EMBEDDING_DIM: int = 1024

    # LLM (Gemini)
    GOOGLE_API_KEY: str = ""
    GEMINI_MODEL: str = "gemini-3.1-flash-lite"

    # Conversation memory
    CONVERSATION_MAX_HISTORY: int = 10
    # ประวัติเก็บในหน่วยความจำของ process ถ้าไม่กวาดทิ้ง แรมจะโตตามจำนวน session ที่เคยเข้ามา
    # จนโดน OOM kill (โมเดล embedding กินไปแล้วราว 2.3GB ดู mem_limit ของ ai-service-app)
    # CONVERSATION_TTL_SECONDS: session ที่เงียบเกินเวลานี้ถูกลบทิ้ง
    # CONVERSATION_MAX_SESSIONS: เพดานจำนวน session ที่ถือไว้พร้อมกัน เกินแล้วไล่ตัวที่เก่าสุดออกก่อน
    CONVERSATION_TTL_SECONDS: int = 3600
    CONVERSATION_MAX_SESSIONS: int = 1000

    # RAG retrieval
    # RAG_TOP_K: จำนวนสินค้าสูงสุดที่ดึงเข้า context prompt (แลกระหว่าง recall กับ token)
    # RAG_SCORE_THRESHOLD: cosine similarity ขั้นต่ำ (collection ใช้ Distance.COSINE)
    #   ค่านี้มาจากการกวาดหา threshold ที่แยกคำถามในแคตตาล็อกออกจากคำถามนอกแคตตาล็อกได้ครบ
    #   ผูกกับ BAAI/bge-m3 เท่านั้น ถ้าเปลี่ยน EMBEDDING_MODEL ต้องกวาดใหม่ (สเกลคะแนนต่างกัน)
    RAG_TOP_K: int = 5
    RAG_SCORE_THRESHOLD: float = 0.45

    class Config:
        env_file = ".env"


settings = Settings()
