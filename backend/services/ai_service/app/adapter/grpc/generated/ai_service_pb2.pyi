from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ChatRequest(_message.Message):
    __slots__ = ("message", "session_id")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    message: str
    session_id: str
    def __init__(self, message: _Optional[str] = ..., session_id: _Optional[str] = ...) -> None: ...

class ChatResponse(_message.Message):
    __slots__ = ("event_type", "text_content", "product_ids")
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    TEXT_CONTENT_FIELD_NUMBER: _ClassVar[int]
    PRODUCT_IDS_FIELD_NUMBER: _ClassVar[int]
    event_type: str
    text_content: str
    product_ids: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, event_type: _Optional[str] = ..., text_content: _Optional[str] = ..., product_ids: _Optional[_Iterable[int]] = ...) -> None: ...
