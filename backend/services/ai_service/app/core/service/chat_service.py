"""Chat service — RAG business logic using LangChain."""

import logging
import re
from collections import defaultdict
from typing import AsyncGenerator

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from langchain_core.output_parsers import StrOutputParser
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
        self._prompt = ChatPromptTemplate.from_messages([
            ("system", SYSTEM_TEMPLATE),
            MessagesPlaceholder(variable_name="chat_history"),
            ("human", "{question}"),
        ])

        # RAG chain: prompt -> LLM -> parse output
        self._chain = self._prompt | self._llm

        # Contextualize query prompt
        self._contextualize_prompt = ChatPromptTemplate.from_messages([
            ("system", CONTEXTUALIZE_TEMPLATE),
            MessagesPlaceholder(variable_name="chat_history"),
            ("human", "{question}"),
        ])
        self._contextualize_chain = self._contextualize_prompt | self._rewriter_llm | StrOutputParser()

        # In-memory conversation history per session
        self._history: dict[str, list] = defaultdict(list)

    async def chat(self, request: ChatRequest) -> AsyncGenerator[ChatResponseChunk, None]:
        # 0. Contextualize user query using chat history
        chat_history = self._history[request.session_id]
        search_query = request.message
        
        if chat_history:
            try:
                rewritten = await self._contextualize_chain.ainvoke({
                    "chat_history": chat_history[-REWRITE_HISTORY_MSGS:],
                    "question": request.message
                })
                rewritten = rewritten.strip()
                if rewritten:
                    search_query = rewritten
                logger.info("Contextualized query: %s", search_query)
            except Exception:
                logger.exception("Error contextualizing query")

        # 1-2. Embed the query (or queries) and retrieve products
        products: list[ProductResult] = await self._retrieve(request.message, search_query)

        # 3. Build RAG context
        context = self._build_context(products)
        
        reply_content = ""

        # 5. Invoke LangChain chain
        try:
            async for chunk in self._chain.astream({
                "context": context,
                "chat_history": chat_history,
                "question": request.message,
            }):
                if chunk.content:
                    reply_content += chunk.content
                    yield ChatResponseChunk(event_type="chunk", text_content=chunk.content)
            
            # 5.1 Yield products that were actually mentioned by the AI
            relevant_product_ids = self._mentioned_product_ids(products, reply_content)

            if relevant_product_ids:
                yield ChatResponseChunk(event_type="products", product_ids=relevant_product_ids)
            
            yield ChatResponseChunk(event_type="done")
        except Exception:
            logger.exception("LangChain chain error")
            
            # Yield any products mentioned before the crash
            relevant_product_ids = self._mentioned_product_ids(products, reply_content)
            if relevant_product_ids:
                yield ChatResponseChunk(event_type="products", product_ids=relevant_product_ids)

            yield ChatResponseChunk(event_type="error", text_content="ขออภัยค่ะ ระบบ AI มีปัญหาชั่วคราว กรุณาลองใหม่อีกครั้งนะคะ")
            reply_content += "\n[error]"

        # 6. Update conversation history
        chat_history.append(HumanMessage(content=request.message))
        chat_history.append(AIMessage(content=reply_content))

        # Trim history to prevent token overflow
        max_msgs = settings.CONVERSATION_MAX_HISTORY * 2
        if len(chat_history) > max_msgs:
            self._history[request.session_id] = chat_history[-max_msgs:]

    async def _retrieve(self, raw_query: str, search_query: str) -> list[ProductResult]:
        """ค้นด้วยคำถามที่เขียนใหม่ และค้นด้วยคำถามดิบที่ผู้ใช้พิมพ์ควบคู่กันไป

        ต่อให้ตัวเขียนคำถามใหม่ลากเงื่อนไขเทิร์นก่อนมาปนอีก สินค้าที่ตรงกับสิ่งที่ผู้ใช้
        พิมพ์จริงก็ยังเข้า context ต้นทุนคือ embed เพิ่มหนึ่งครั้ง ซึ่งรันด้วยโมเดลในเครื่องอยู่แล้ว
        """
        queries = [search_query]
        if raw_query.strip() and raw_query.strip() != search_query.strip():
            queries.append(raw_query)

        ranked_lists: list[list[ProductResult]] = []
        for query in queries:
            hits = await self._vector_store.search(
                query_embedding=self._embedding.embed(query),
                top_k=settings.RAG_TOP_K,
                score_threshold=settings.RAG_SCORE_THRESHOLD,
            )
            logger.info("Retrieved %d for %r: %s", len(hits), query, self._format_hits(hits))
            ranked_lists.append(hits)

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
            if tokens and all(t in haystack for t in tokens):
                ids.append(p.product_id)
        return ids

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
            product_info = f"{i}. {p.name} — {p.description} (ราคาเริ่มต้น {p.min_price:.2f} บาท)"
            
            # Format variants if available
            if p.variants:
                variant_lines = []
                for v in p.variants:
                    v_name = v.get("name", "")
                    v_price = v.get("price", 0.0)
                    v_attrs = v.get("attributes", {})
                    attr_str = ", ".join(f"{k}: {val}" for k, val in v_attrs.items())
                    variant_info = f"    - ตัวเลือก: {v_name} (ราคา {v_price:.2f} บาท)"
                    if attr_str:
                        variant_info += f" [{attr_str}]"
                    variant_lines.append(variant_info)
                product_info += "\n" + "\n".join(variant_lines)
            
            lines.append(product_info)
            
        return "\n".join(lines)
