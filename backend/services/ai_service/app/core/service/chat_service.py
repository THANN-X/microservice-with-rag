"""Chat service — RAG business logic using LangChain."""

import logging
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
            ("system", "Given a chat history and the latest user question which might reference context in the chat history, formulate a standalone question which can be understood without the chat history. Do NOT answer the question, just reformulate it if needed and otherwise return it as is."),
            MessagesPlaceholder(variable_name="chat_history"),
            ("human", "{question}"),
        ])
        self._contextualize_chain = self._contextualize_prompt | self._llm | StrOutputParser()

        # In-memory conversation history per session
        self._history: dict[str, list] = defaultdict(list)

    async def chat(self, request: ChatRequest) -> AsyncGenerator[ChatResponseChunk, None]:
        # 0. Contextualize user query using chat history
        chat_history = self._history[request.session_id]
        search_query = request.message
        
        if chat_history:
            try:
                search_query = await self._contextualize_chain.ainvoke({
                    "chat_history": chat_history,
                    "question": request.message
                })
                logger.info("Contextualized query: %s", search_query)
            except Exception:
                logger.exception("Error contextualizing query")

        # 1. Embed user query
        query_embedding = self._embedding.embed(search_query)

        # 2. Retrieve relevant products from vector store
        products: list[ProductResult] = await self._vector_store.search(
            query_embedding=query_embedding,
            top_k=15,
            score_threshold=0.4,
        )

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
            relevant_product_ids = []
            for p in products:
                if p.name in reply_content:
                    relevant_product_ids.append(p.product_id)
            
            if relevant_product_ids:
                yield ChatResponseChunk(event_type="products", product_ids=relevant_product_ids)
            
            yield ChatResponseChunk(event_type="done")
        except Exception:
            logger.exception("LangChain chain error")
            
            # Yield any products mentioned before the crash
            relevant_product_ids = []
            for p in products:
                if p.name in reply_content:
                    relevant_product_ids.append(p.product_id)
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
