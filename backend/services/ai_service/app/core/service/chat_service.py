"""Chat service — RAG business logic using LangChain."""

import asyncio
import logging
import re
import time
from collections import OrderedDict
from dataclasses import dataclass, field
from typing import AsyncGenerator

from langchain_core.messages import AIMessage, HumanMessage
from langchain_core.output_parsers import StrOutputParser
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from langchain_google_genai import ChatGoogleGenerativeAI

from app.config import settings
from app.core.domain.models import (
    ChatRequest,
    ChatResponseChunk,
    ProductResult,
)
from app.core.port.embedding_port import EmbeddingPort
from app.core.port.vector_store_port import VectorStorePort

logger = logging.getLogger(__name__)

SYSTEM_TEMPLATE = (
    "คุณเป็นผู้ช่วย AI ของร้านค้าออนไลน์ 'อนันตา' "
    "ตอบคำถามเกี่ยวกับสินค้าโดยอ้างอิงจากข้อมูลสินค้าที่ให้มาเท่านั้น "
    "ถ้าไม่มีข้อมูลสินค้าที่เกี่ยวข้อง หรือสินค้าที่ให้มาไม่ตรงกับที่ผู้ใช้ต้องการเลย ให้บอกอย่างสุภาพว่าไม่พบสินค้า และ **ห้าม** เสนอชื่อสินค้าที่ไม่เกี่ยวข้องเด็ดขาด "
    "ตอบเป็นภาษาไทย สุภาพ กระชับ และเป็นมิตร\n\n"
    "--- ข้อมูลสินค้าจากระบบ ---\n{context}"
)


# prompt ตัวเดิมเป็นตัวอย่างมาตรฐานของ LangChain ซึ่งไม่ได้พูดถึงการเปลี่ยนหัวข้อเลย
# ผลคือพอผู้ใช้ถามมือถือแล้วสลับไปถามเสื้อผ้า มันเขียนคำถามใหม่เป็น
# "มีสมาร์ทโฟนสีเขียวไม่เกิน 50,000 หรือทางร้านมีเสื้อผ้าด้วยหรือไม่" แล้วค้นได้แต่มือถือ
# กฎข้อ 1 กับ 3 และตัวอย่างคู่แรกมีไว้กันเคสนี้โดยเฉพาะ
CONTEXTUALIZE_TEMPLATE = (
    "You rewrite the user's latest question into a standalone search query "
    "for an online store's product catalog.\n"
    "Rules:\n"
    "1. If the latest question already makes sense on its own, return it EXACTLY as "
    "written. Do not rephrase it, expand it, or make it more polite.\n"
    "2. Pull details from the chat history ONLY when the latest question depends on "
    "them (e.g. 'แล้วสีอื่นมีไหม', 'อันไหนถูกกว่ากัน').\n"
    "3. When the user moves to a new topic or a different product category, DROP every "
    "constraint from earlier turns — category, colour, budget, brand. "
    "Never merge the previous question with the new one.\n"
    "4. Output one short question and nothing else. No answer, no explanation.\n\n"
    "Examples:\n"
    "History: the user asked for green phones under 50,000 baht.\n"
    "Latest: 'ร้านมีเสื้อผ้าขายไหม' -> 'ร้านมีเสื้อผ้าขายไหม'\n"
    "History: the user asked about noise-cancelling headphones.\n"
    "Latest: 'แล้วสีอื่นมีไหม' -> 'หูฟังตัดเสียงรบกวนมีสีอื่นไหม'"
)

# ส่งเข้าตัวเขียน query ใหม่แค่ 3 เทิร์นล่าสุด คำตอบเก่าๆ ที่ยาวและเต็มไปด้วยชื่อสินค้า
# ดึงคำถามใหม่ให้เอนไปหาหมวดเดิม ส่วน chain ที่ตอบผู้ใช้ยังเห็นประวัติเต็มตามเดิม
REWRITE_HISTORY_MSGS = 6

# markdown กับช่องว่างที่ Gemini แทรกกลางชื่อสินค้า ต้องตัดออกก่อนเทียบชื่อ
_MARKUP = re.compile(r"[*_`#~\s]+")

# token ที่สั้นกว่านี้ในชื่อสินค้า ("5", "XL") ต้องเจอแบบติดกับ token ข้างเคียง ไม่ใช่โผล่ที่ไหนก็ได้
MIN_STANDALONE_TOKEN_LEN = 3


@dataclass
class _Session:
    """ประวัติของ session หนึ่ง พร้อมเวลาที่ถูกใช้ล่าสุดไว้ให้ตัวกวาดใช้ตัดสิน"""

    messages: list = field(default_factory=list)
    last_active: float = 0.0


class ChatService:
    def __init__(
        self,
        vector_store: VectorStorePort,
        embedding: EmbeddingPort,
    ):
        self._vector_store = vector_store
        self._embedding = embedding

        # LangChain LLM
        self._llm = ChatGoogleGenerativeAI(
            model=settings.GEMINI_MODEL,
            google_api_key=settings.GOOGLE_API_KEY,
            temperature=0.7,
        )

        # ตัวเขียน query ใหม่ต้องนิ่ง ใช้ LLM ตัวเดียวกับที่ตอบคำถาม (temperature 0.7) ไม่ได้
        # เพราะคำถามเดิมจะได้ query คนละแบบทุกครั้ง จนรอบแรกค้นไม่เจอ รอบสองเจอ โดยไม่มีอะไรเปลี่ยน
        self._rewriter_llm = ChatGoogleGenerativeAI(
            model=settings.GEMINI_MODEL,
            google_api_key=settings.GOOGLE_API_KEY,
            temperature=0.0,
        )

        # Prompt with conversation history
        self._prompt = ChatPromptTemplate.from_messages(
            [
                ("system", SYSTEM_TEMPLATE),
                MessagesPlaceholder(variable_name="chat_history"),
                ("human", "{question}"),
            ]
        )

        # RAG chain: prompt -> LLM -> parse output
        self._chain = self._prompt | self._llm

        # Contextualize query prompt
        self._contextualize_prompt = ChatPromptTemplate.from_messages(
            [
                ("system", CONTEXTUALIZE_TEMPLATE),
                MessagesPlaceholder(variable_name="chat_history"),
                ("human", "{question}"),
            ]
        )
        self._contextualize_chain = (
            self._contextualize_prompt | self._rewriter_llm | StrOutputParser()
        )

        # In-memory conversation history per session (เรียงจากตัวที่ไม่ได้ใช้นานสุด -> ตัวล่าสุด)
        self._history: OrderedDict[str, _Session] = OrderedDict()

    def _session_history(self, session_id: str) -> list:
        """คืน list ประวัติของ session พร้อมกวาดตัวที่หมดอายุและคุมจำนวน session ไม่ให้โตไม่หยุด

        session_id มาจาก client ตรงๆ ถ้าเก็บทุกตัวไว้ตลอดอายุ process แรมจะโตทางเดียวจนโดน OOM kill
        (ยิง session_id สุ่มรัวๆ ก็เร่งให้ถึงจุดนั้นได้)
        """
        now = time.monotonic()
        self._purge_expired(now)

        session = self._history.get(session_id)
        if session is None:
            session = _Session()
            self._history[session_id] = session
        session.last_active = now
        self._history.move_to_end(session_id)

        # ตัวที่เพิ่งถูกแตะย้ายไปอยู่ท้ายสุดแล้ว จึงไม่มีทางโดนไล่ออกเอง
        while len(self._history) > settings.CONVERSATION_MAX_SESSIONS:
            evicted, _ = self._history.popitem(last=False)
            logger.info("Evicted least-recently-used chat session %s", evicted)

        return session.messages

    def _purge_expired(self, now: float) -> None:
        ttl = settings.CONVERSATION_TTL_SECONDS
        while self._history:
            # เรียงตามการใช้งานอยู่แล้ว เจอตัวแรกที่ยังไม่หมดอายุก็หยุดได้เลย
            session_id, session = next(iter(self._history.items()))
            if now - session.last_active < ttl:
                break
            del self._history[session_id]
            logger.info("Expired idle chat session %s (idle > %ds)", session_id, ttl)

    async def chat(self, request: ChatRequest) -> AsyncGenerator[ChatResponseChunk, None]:
        # 0. Contextualize user query using chat history
        chat_history = self._session_history(request.session_id)
        search_query = request.message

        if chat_history:
            try:
                rewritten = await self._contextualize_chain.ainvoke(
                    {
                        "chat_history": chat_history[-REWRITE_HISTORY_MSGS:],
                        "question": request.message,
                    }
                )
                rewritten = rewritten.strip()
                if rewritten:
                    search_query = rewritten
                logger.info("Contextualized query: %s", search_query)
            except Exception:
                logger.exception("Error contextualizing query")

        # 1-3. Embed the query (or queries), retrieve products, build RAG context
        # แยก try ของตัวเองเพราะค้นหาล่มคนละสาเหตุกับ LLM ล่ม และควรมีข้อความแจ้งของตัวเอง
        # ตกลงไปตอบด้วย context ว่างไม่ได้ เพราะ system prompt จะสั่งให้ตอบว่า "ไม่พบสินค้า"
        # เท่ากับบอกผู้ใช้เต็มปากว่าร้านไม่มีของ ทั้งที่แค่ Qdrant ล่มชั่วคราว
        try:
            products: list[ProductResult] = await self._retrieve(request.message, search_query)
            context = self._build_context(products)
        except Exception:
            logger.exception("Error retrieving products")
            yield ChatResponseChunk(
                event_type="error",
                text_content="ขออภัยค่ะ ระบบค้นหาสินค้าขัดข้องชั่วคราว กรุณาลองใหม่อีกครั้งนะคะ",
            )
            yield ChatResponseChunk(event_type="done")
            return

        reply_content = ""
        failed = False

        # 5. Invoke LangChain chain
        try:
            async for chunk in self._chain.astream(
                {
                    "context": context,
                    "chat_history": chat_history,
                    "question": request.message,
                }
            ):
                if chunk.content:
                    reply_content += chunk.content
                    yield ChatResponseChunk(event_type="chunk", text_content=chunk.content)

            # 5.1 Yield products that were actually mentioned by the AI
            relevant_product_ids = self._mentioned_product_ids(products, reply_content)

            if relevant_product_ids:
                yield ChatResponseChunk(event_type="products", product_ids=relevant_product_ids)

            yield ChatResponseChunk(event_type="done")
        except Exception:
            failed = True
            logger.exception("LangChain chain error")

            # Yield any products mentioned before the crash
            relevant_product_ids = self._mentioned_product_ids(products, reply_content)
            if relevant_product_ids:
                yield ChatResponseChunk(event_type="products", product_ids=relevant_product_ids)

            yield ChatResponseChunk(
                event_type="error", text_content="ขออภัยค่ะ ระบบ AI มีปัญหาชั่วคราว กรุณาลองใหม่อีกครั้งนะคะ"
            )
        finally:
            # 6. Update conversation history
            # ต้องอยู่ใน finally เพราะถ้าผู้ใช้ปิดแท็บกลางคัน generator จะถูกปิดที่บรรทัด yield
            # โค้ดที่อยู่ถัดจาก try จะไม่ถูกรันเลย เทิร์นนั้นหายไปจากประวัติทั้งที่ผู้ใช้เห็นคำตอบไปแล้ว
            # แต่ไม่บันทึกเมื่อ error หรือคำตอบว่าง — ข้อความพังจะถูกส่งกลับเข้าโมเดลในเทิร์นถัดไป
            # ส่วนคำตอบที่ขาดเพราะผู้ใช้ปิดแท็บยังบันทึก เพราะเป็นสิ่งที่ผู้ใช้เห็นจริงและใช้อ้างอิงต่อได้
            if not failed and reply_content.strip():
                chat_history.append(HumanMessage(content=request.message))
                chat_history.append(AIMessage(content=reply_content))

                # Trim history to prevent token overflow
                # ตัดในที่ เพื่อให้ยังเป็น list เดียวกับที่ _Session ถืออยู่
                max_msgs = settings.CONVERSATION_MAX_HISTORY * 2
                if len(chat_history) > max_msgs:
                    del chat_history[:-max_msgs]

    async def _retrieve(self, raw_query: str, search_query: str) -> list[ProductResult]:
        """ค้นด้วยคำถามที่เขียนใหม่ และค้นด้วยคำถามดิบที่ผู้ใช้พิมพ์ควบคู่กันไป

        ต่อให้ตัวเขียนคำถามใหม่ลากเงื่อนไขเทิร์นก่อนมาปนอีก สินค้าที่ตรงกับสิ่งที่ผู้ใช้
        พิมพ์จริงก็ยังเข้า context ต้นทุนคือ embed เพิ่มหนึ่งคำถาม ซึ่งรวมอยู่ใน batch เดียวกันอยู่แล้ว
        """
        queries = [search_query]
        if raw_query.strip() and raw_query.strip() != search_query.strip():
            queries.append(raw_query)

        # embed เป็นงาน CPU ล้วนและเป็น sync เรียกตรงๆ จะบล็อก event loop ทั้ง process
        # (ทุก gRPC stream, health check และงาน index จาก Kafka อยู่บน loop เดียวกันหมด)
        # ยิงทุกคำถามเป็น batch เดียวใน thread แยก ได้ทั้งไม่บล็อกและเหลือ forward pass เดียว
        embeddings = await asyncio.to_thread(self._embedding.embed_batch, queries)

        ranked_lists: list[list[ProductResult]] = list(
            await asyncio.gather(
                *(
                    self._vector_store.search(
                        query_embedding=embedding,
                        top_k=settings.RAG_TOP_K,
                        score_threshold=settings.RAG_SCORE_THRESHOLD,
                    )
                    for embedding in embeddings
                )
            )
        )
        for query, hits in zip(queries, ranked_lists):
            logger.info("Retrieved %d for %r: %s", len(hits), query, self._format_hits(hits))

        products = self._merge_ranked(ranked_lists, settings.RAG_TOP_K)
        if len(ranked_lists) > 1:
            logger.info("Merged context: %s", self._format_hits(products))
        return products

    @staticmethod
    def _merge_ranked(ranked_lists: list[list[ProductResult]], limit: int) -> list[ProductResult]:
        """สลับหยิบทีละอันดับจากทุกรายการ เพื่อให้ทุกคำถามมีที่ใน context เสมอ

        เรียงตามคะแนนล้วนไม่ได้ — คำถามที่ปนเปื้อนมักได้คะแนนสูงกว่าเพราะคำเยอะกว่าและ
        ตรงกับสินค้าหมวดเดิมเป๊ะกว่า แล้วจะเบียดสินค้าที่ผู้ใช้ถามจริงตกไปทั้งหมด
        """
        merged: dict[int, ProductResult] = {}
        order: list[ProductResult] = []
        depth = max((len(lst) for lst in ranked_lists), default=0)
        for rank in range(depth):
            for lst in ranked_lists:
                if rank >= len(lst):
                    continue
                p = lst[rank]
                seen = merged.get(p.product_id)
                if seen is None:
                    merged[p.product_id] = p
                    order.append(p)
                elif p.relevance_score > seen.relevance_score:
                    seen.relevance_score = p.relevance_score
        return order[:limit]

    @staticmethod
    def _mentioned_product_ids(products: list[ProductResult], reply: str) -> list[int]:
        """หาว่า AI พูดถึงสินค้าชิ้นไหนบ้าง เพื่อส่ง id ให้หน้าเว็บแสดงการ์ดสินค้า

        เทียบชื่อเต็มแบบ substring ไม่ได้ผล เพราะ Gemini เรียบเรียงชื่อใหม่เป็นประจำ
        เช่นสินค้าชื่อ "ซีนิท หูฟังไร้สาย" ถูกเขียนในคำตอบว่า "**รุ่นซีนิท**"
        แล้วการ์ดหายทั้งข้อความทั้งที่ AI แนะนำสินค้าครบ
        จึงเช็คว่าทุกคำในชื่อสินค้าโผล่ในคำตอบครบหรือไม่ หลังตัด markdown และช่องว่างออก
        ถ้า AI ตอบว่าไม่พบสินค้า จะไม่มีชื่อแบรนด์ในคำตอบ ผลลัพธ์จึงว่างตามเดิม
        """
        haystack = _MARKUP.sub("", reply)
        ids: list[int] = []
        for p in products:
            tokens = [_MARKUP.sub("", t) for t in p.name.split()]
            tokens = [t for t in tokens if t]
            if tokens and ChatService._name_mentioned(tokens, haystack):
                ids.append(p.product_id)
        return ids

    @staticmethod
    def _name_mentioned(tokens: list[str], haystack: str) -> bool:
        """ทุก token ต้องโผล่ในคำตอบ และ token สั้นต้องโผล่แบบติดกับ token ข้างเคียงของชื่อนั้นจริงๆ

        haystack ถูกตัดช่องว่างออกหมดเพื่อให้ทนกับการที่ Gemini แทรก markdown กลางชื่อ
        ผลข้างเคียงคือ token สั้นแมตช์ข้ามขอบคำได้ง่ายมาก — สินค้า "ซีนิท หูฟัง รุ่น 5"
        จะเด้งการ์ดขึ้นมาทั้งที่ AI แนะนำ "รุ่น 3" ถ้าในคำตอบมีคำว่า "รับประกัน 5 ปี" อยู่ที่ไหนสักแห่ง
        การบังคับให้เจอ "รุ่น5" ติดกัน ตัดเคสนี้ทิ้งโดยไม่ทำให้ชื่อที่ถูกเรียบเรียงใหม่หลุดหาย
        """
        for i, token in enumerate(tokens):
            if token not in haystack:
                return False
            if len(token) >= MIN_STANDALONE_TOKEN_LEN:
                continue
            neighbours = []
            if i > 0:
                neighbours.append(tokens[i - 1] + token)
            if i + 1 < len(tokens):
                neighbours.append(token + tokens[i + 1])
            if neighbours and not any(n in haystack for n in neighbours):
                return False
        return True

    @staticmethod
    def _format_hits(products: list[ProductResult]) -> str:
        if not products:
            return "(ไม่มี)"
        return ", ".join(f"{p.name} {p.relevance_score:.4f}" for p in products)

    @staticmethod
    def _build_context(products: list[ProductResult]) -> str:
        if not products:
            return "ไม่พบสินค้าที่เกี่ยวข้องในระบบ"

        lines = ["สินค้าที่เกี่ยวข้อง:"]
        for i, p in enumerate(products, 1):
            # payload จาก Qdrant อาจมีคีย์อยู่แต่ค่าเป็น None (point ที่เขียนด้วยสคีมาเก่าหรือเครื่องมืออื่น)
            # ซึ่ง .get(k, default) ไม่ช่วย และ f"{None:.2f}" จะโยน TypeError ทำให้ทั้งเทิร์นตาย
            min_price = p.min_price or 0.0
            product_info = f"{i}. {p.name} — {p.description} (ราคาเริ่มต้น {min_price:.2f} บาท)"

            # Format variants if available
            if p.variants:
                variant_lines = []
                for v in p.variants:
                    v_name = v.get("name") or ""
                    v_price = v.get("price") or 0.0
                    v_attrs = v.get("attributes") or {}
                    attr_str = ", ".join(f"{k}: {val}" for k, val in v_attrs.items())
                    variant_info = f"    - ตัวเลือก: {v_name} (ราคา {v_price:.2f} บาท)"
                    if attr_str:
                        variant_info += f" [{attr_str}]"
                    variant_lines.append(variant_info)
                product_info += "\n" + "\n".join(variant_lines)

            lines.append(product_info)

        return "\n".join(lines)
