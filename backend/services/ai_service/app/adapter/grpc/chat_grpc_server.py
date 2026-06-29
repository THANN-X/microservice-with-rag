"""gRPC server adapter — handles incoming gRPC requests from BFF."""

import logging
from concurrent import futures

import grpc

from app.adapter.grpc.generated import ai_service_pb2, ai_service_pb2_grpc
from app.core.domain.models import ChatRequest
from app.core.service.chat_service import ChatService

logger = logging.getLogger(__name__)


class AIChatServicer(ai_service_pb2_grpc.AIChatServiceServicer):
    def __init__(self, chat_service: ChatService):
        self._chat_service = chat_service

    async def Chat(self, request, context):
        try:
            chat_req = ChatRequest(
                message=request.message,
                session_id=request.session_id,
            )
            
            async for chunk in self._chat_service.chat(chat_req):
                yield ai_service_pb2.ChatResponse(
                    event_type=chunk.event_type,
                    text_content=chunk.text_content,
                    product_ids=chunk.product_ids,
                )
        except Exception:
            logger.exception("Error in gRPC Chat handler")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("Internal server error")
            yield ai_service_pb2.ChatResponse(event_type="error", text_content="ขออภัยค่ะ เกิดข้อผิดพลาด")


async def serve_grpc(chat_service: ChatService, port: int) -> grpc.aio.Server:
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    servicer = AIChatServicer(chat_service)
    ai_service_pb2_grpc.add_AIChatServiceServicer_to_server(servicer, server)
    server.add_insecure_port(f"[::]:{port}")
    await server.start()
    logger.info("gRPC server started on port %d", port)
    return server
