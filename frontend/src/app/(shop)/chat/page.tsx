"use client";

import { useState, useRef, useEffect } from "react";
import Image from "next/image";
import ReactMarkdown from "react-markdown";
import Link from "next/link";
import { formatBaht, getMinPrice } from "@/lib/utils";
import { catalogService } from "@/lib/services";
import { CatalogProduct } from "@/lib/types";

interface Message {
  id: number;
  role: "user" | "ai";
  text: string;
  productIds?: number[];
}

function ProductRecommendations({ productIds }: { productIds: number[] }) {
  const [products, setProducts] = useState<CatalogProduct[]>([]);

  useEffect(() => {
    Promise.all(productIds.map((id) => catalogService.get(String(id))))
      .then((responses) => {
        // filter out nulls or errors and extract product data
        const validProducts = responses
          .map((r: any) => r?.data || r)
          .filter((p: any) => p && (p.id || p.product_id));
        setProducts(validProducts as CatalogProduct[]);
      })
      .catch(console.error);
  }, [productIds]);

  if (!products.length) return null;

  return (
    <div className="flex gap-4 overflow-x-auto py-2 mt-1 pb-4 max-w-[80vw] md:max-w-[45vw] lg:max-w-[35vw] scroll-smooth snap-x">
      {products.map((p) => {
        const id = p.product_id || p.id;
        return (
          <Link
            key={id}
            href={`/products/${id}`}
            className="min-w-[160px] w-[160px] bg-white rounded-xl p-3 hover:shadow-md transition-shadow shrink-0 block border border-outline-variant/20 snap-start"
          >
            <div className="relative aspect-[4/3] rounded-lg overflow-hidden bg-surface-container mb-2">
              {p.image_urls?.[0] ? (
                <Image src={p.image_urls[0]} alt={p.name} fill className="object-cover" />
              ) : (
                <div className="w-full h-full flex items-center justify-center">
                  <span className="material-symbols-outlined text-outline">image</span>
                </div>
              )}
            </div>
            <h4 className="text-xs font-bold line-clamp-1 mb-1">{p.name}</h4>
            <span className="text-primary font-black text-sm">{formatBaht(getMinPrice(p.variants))}</span>
          </Link>
        );
      })}
    </div>
  );
}

export default function ChatPage() {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: 1,
      role: "ai",
      text: "สวัสดีค่ะ! ดิฉันเป็นผู้ช่วย AI ของอนันตา ยินดีให้บริการค่ะ ต้องการสอบถามเรื่องอะไรคะ? ไม่ว่าจะเป็นการค้นหาสินค้า เปรียบเทียบราคา หรือขอคูปองส่วนลด สามารถถามได้เลยค่ะ!",
    },
  ]);
  const [input, setInput] = useState("");
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const sessionIdRef = useRef<string>("");
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    const saved = sessionStorage.getItem("chat_messages");
    if (saved) {
      try {
        setMessages(JSON.parse(saved));
      } catch (e) { }
    }
    const savedSessionId = sessionStorage.getItem("chat_session_id");
    if (savedSessionId) {
      sessionIdRef.current = savedSessionId;
    } else {
      sessionIdRef.current = Math.random().toString(36).substring(2, 15);
      sessionStorage.setItem("chat_session_id", sessionIdRef.current);
    }
    setIsLoaded(true);
  }, []);

  useEffect(() => {
    if (isLoaded) {
      sessionStorage.setItem("chat_messages", JSON.stringify(messages));
    }
  }, [messages, isLoaded]);

  useEffect(() => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTo({
        top: scrollContainerRef.current.scrollHeight,
        behavior: "smooth",
      });
    }
  }, [messages.length]);

  const handleSend = async () => {
    if (!input.trim()) return;
    const userInput = input.trim();
    setInput("");

    if (!sessionIdRef.current) {
      sessionIdRef.current = Math.random().toString(36).substring(2, 15);
    }
    const userMsg: Message = { id: Date.now(), role: "user", text: userInput };
    setMessages((prev) => [...prev, userMsg]);

    try {
      const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || "http://localhost:8080";
      const res = await fetch(`${BFF_URL}/chat`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: userInput, session_id: sessionIdRef.current }),
      });

      if (!res.ok || !res.body) throw new Error("Failed to connect to chat API");

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let done = false;

      const aiMsgId = Date.now() + 1;
      setMessages((prev) => [...prev, { id: aiMsgId, role: "ai", text: "" }]);
      let aiText = "";
      let buffer = "";

      while (!done) {
        const { value, done: readerDone } = await reader.read();
        done = readerDone;
        if (value) {
          buffer += decoder.decode(value, { stream: true });
          let eolIndex;
          // SSE messages are separated by \n\n
          while ((eolIndex = buffer.indexOf('\n\n')) >= 0) {
            const messageStr = buffer.slice(0, eolIndex);
            buffer = buffer.slice(eolIndex + 2);

            const lines = messageStr.split('\n');
            let dataStr = "";
            for (const line of lines) {
              if (line.startsWith("data: ")) {
                dataStr = line.slice(6);
              }
            }
            if (dataStr) {
              try {
                const data = JSON.parse(dataStr);
                if (data.event_type === "chunk" || data.event_type === "done") {
                  aiText += (data.text_content || "");
                  setMessages((prev) =>
                    prev.map((m) => (m.id === aiMsgId ? { ...m, text: aiText } : m))
                  );
                } else if (data.event_type === "products") {
                  setMessages((prev) =>
                    prev.map((m) => (m.id === aiMsgId ? { ...m, productIds: data.product_ids } : m))
                  );
                } else if (data.event_type === "error") {
                  aiText += "\n[เกิดข้อผิดพลาด: " + (data.text_content || "") + "]";
                  setMessages((prev) =>
                    prev.map((m) => (m.id === aiMsgId ? { ...m, text: aiText } : m))
                  );
                }
              } catch (e) {
                console.error("Error parsing JSON data", e);
              }
            }
          }
        }
      }
    } catch (error) {
      console.error(error);
      const aiMsg: Message = {
        id: Date.now() + 1,
        role: "ai",
        text: "ขออภัยค่ะ ระบบขัดข้องชั่วคราว กรุณาลองใหม่อีกครั้งนะคะ",
      };
      setMessages((prev) => [...prev, aiMsg]);
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-7rem)] -mx-5 md:-mx-10 -mb-24 relative">
      {/* Chat Header */}
      <div className="bg-surface-container-lowest px-6 py-4 border-b border-outline-variant/10 flex items-center gap-3.5">
        <div className="w-10 h-10 rounded-full editorial-gradient flex items-center justify-center text-white">
          <span className="material-symbols-outlined text-xl" style={{ fontVariationSettings: "'FILL' 1" }}>
            smart_toy
          </span>
        </div>
        <div>
          <h1 className="font-bold text-on-surface text-sm">AI ผู้ช่วยอัจฉริยะ</h1>
          <span className="text-[10px] text-green-600 flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
            กำลังออนไลน์
          </span>
        </div>
      </div>

      {/* Messages */}
      <div ref={scrollContainerRef} className="flex-1 overflow-y-auto px-6 py-5 space-y-5 no-scrollbar">
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
          >
            {msg.role === "ai" && (
              <div className="w-7 h-7 rounded-full bg-primary flex items-center justify-center text-white mr-2.5 shrink-0 mt-1">
                <span className="material-symbols-outlined text-xs" style={{ fontVariationSettings: "'FILL' 1" }}>
                  smart_toy
                </span>
              </div>
            )}
            <div className={`flex flex-col gap-1 ${msg.role === "user" ? "items-end" : "items-start"} max-w-[85%]`}>
              <div
                className={`px-4 py-3 rounded-2xl text-sm leading-relaxed shadow-sm ${msg.role === "user"
                  ? "bg-primary text-white rounded-br-sm"
                  : "bg-surface border border-outline-variant/20 text-on-surface rounded-bl-sm"
                  }`}
              >
                {msg.role === "user" ? (
                  msg.text
                ) : (
                  <ReactMarkdown
                    components={{
                      p: ({ node, ...props }) => <p className="mb-2 last:mb-0" {...props} />,
                      strong: ({ node, ...props }) => <strong className="font-bold text-primary" {...props} />,
                      ul: ({ node, ...props }) => <ul className="list-disc pl-5 mb-2 space-y-1" {...props} />,
                      ol: ({ node, ...props }) => <ol className="list-decimal pl-5 mb-2 space-y-1" {...props} />,
                      li: ({ node, ...props }) => <li className="mb-1" {...props} />,
                    }}
                  >
                    {msg.text}
                  </ReactMarkdown>
                )}
              </div>
              {msg.role === "ai" && msg.productIds && msg.productIds.length > 0 && (
                <ProductRecommendations productIds={msg.productIds} />
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Input Area */}
      <div className="p-4 bg-surface/80 backdrop-blur-xl border-t border-outline-variant/10">
        <div className="max-w-4xl mx-auto flex items-center gap-3 bg-surface-container-highest rounded-xl px-4 py-2.5 border border-transparent focus-within:border-primary/20 focus-within:bg-surface-container-lowest transition-all">
          <input
            className="flex-1 bg-transparent border-none focus:ring-0 text-sm text-on-surface py-1.5 leading-relaxed outline-none"
            placeholder="พิมพ์ข้อความคุยกับ AI..."
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSend()}
          />
          <button
            onClick={handleSend}
            disabled={!input.trim()}
            className="bg-primary text-white p-2 rounded-lg flex items-center justify-center hover:bg-primary-dim active:scale-95 transition-all disabled:opacity-40"
          >
            <span className="material-symbols-outlined text-lg">send</span>
          </button>
        </div>
        <p className="text-center text-[10px] text-on-surface-variant mt-2.5">
          ถามเกี่ยวกับสินค้า การจัดส่ง หรือขอคูปองส่วนลดได้เลย
        </p>
      </div>
    </div>
  );
}
