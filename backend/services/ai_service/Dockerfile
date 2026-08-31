FROM python:3.12-slim AS builder

WORKDIR /app

# Install system dependencies for building
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY backend/services/ai_service/requirements.txt .

# ลง torch แบบ CPU-only ก่อน requirements
# WHY: sentence-transformers ดึง torch มาเป็น dependency และ wheel ปกติบน PyPI
# พ่วง CUDA runtime มาด้วยราว 2.5GB ทั้งที่เรารัน inference บน CPU ล้วน
# ลงจาก index CPU ก่อน ทำให้ pip เห็นว่า torch ถูกติดตั้งแล้วและไม่ไปดึงตัว CUDA มาทับ
# ผลคือ image เล็กลงจาก ~8.8GB เหลือระดับ 2-3GB และ build เร็วขึ้นมาก
RUN pip install --no-cache-dir torch --index-url https://download.pytorch.org/whl/cpu

RUN pip install --no-cache-dir -r requirements.txt

COPY backend/services/ai_service/ .

# Generate gRPC stubs from proto
RUN python -m app.generate_proto

FROM python:3.12-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin
COPY --from=builder /app .

EXPOSE 50051 8000

CMD ["python", "-m", "app.main"]
