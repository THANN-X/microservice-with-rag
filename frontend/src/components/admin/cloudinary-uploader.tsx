// WHAT: อัปโหลดไฟล์รูปหลายรูปพร้อมกันไปยัง Cloudinary แล้วเก็บ URL กลับเป็น string[]
// WHY Cloudinary: ไม่ต้อง host รูปเอง — ได้ CDN + auto-resize URL ฟรี
"use client";

import { useState } from "react";
import Image from "next/image";
import { Upload, X } from "lucide-react";
import { cn } from "@/lib/utils";

const CLOUDINARY_CLOUD_NAME = process.env.NEXT_PUBLIC_CLOUDINARY_CLOUD_NAME ?? "";
const CLOUDINARY_UPLOAD_PRESET = process.env.NEXT_PUBLIC_CLOUDINARY_UPLOAD_PRESET ?? "";

export function CloudinaryImageUploader({
  urls,
  onChange,
}: {
  urls: string[];
  onChange: (urls: string[]) => void;
}) {
  const [uploading, setUploading] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");

  const handleFiles = async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    setUploading(true);
    setErrorMsg("");
    try {
      // อัปโหลดทุกไฟล์พร้อมกัน (parallel) — Cloudinary คืน secure_url (HTTPS) ต่อไฟล์
      const uploaded = await Promise.all(
        Array.from(files).map(async (file) => {
          const fd = new FormData();
          fd.append("file", file);
          fd.append("upload_preset", CLOUDINARY_UPLOAD_PRESET);
          const res = await fetch(
            `https://api.cloudinary.com/v1_1/${CLOUDINARY_CLOUD_NAME}/image/upload`,
            { method: "POST", body: fd }
          );
          if (!res.ok) throw new Error("Upload failed");
          const data = (await res.json()) as { secure_url: string };
          return data.secure_url;
        })
      );
      // append URL ใหม่ต่อท้าย URL เดิม (ไม่แทนที่)
      onChange([...urls, ...uploaded]);
    } catch {
      setErrorMsg("อัปโหลดไม่สำเร็จ กรุณาลองใหม่อีกครั้ง");
    } finally {
      setUploading(false);
    }
  };

  const remove = (idx: number) => onChange(urls.filter((_, i) => i !== idx));

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-2">
        {urls.map((url, i) => (
          <div key={i} className="relative h-16 w-16">
            <Image src={url} alt="" fill className="rounded-lg object-cover bg-surface-highest" sizes="64px" />
            <button
              type="button"
              onClick={() => remove(i)}
              className="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-error text-white shadow-sm z-10"
            >
              <X size={10} />
            </button>
          </div>
        ))}
        <label
          className={cn(
            "flex h-16 w-16 cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-surface-highest text-outline transition-colors hover:border-primary/40 hover:text-primary",
            uploading && "cursor-not-allowed opacity-50"
          )}
        >
          {uploading ? (
            <span className="px-1 text-center text-[9px] leading-tight">กำลังอัปโหลด...</span>
          ) : (
            <>
              <Upload size={16} />
              <span className="mt-0.5 text-[9px]">อัปโหลด</span>
            </>
          )}
          <input
            type="file"
            accept="image/*"
            multiple
            disabled={uploading}
            className="hidden"
            onChange={(e) => handleFiles(e.target.files)}
          />
        </label>
      </div>
      {!CLOUDINARY_CLOUD_NAME && (
        <p className="text-[10px] text-amber-600">
          ⚠ กรุณาตั้งค่า NEXT_PUBLIC_CLOUDINARY_CLOUD_NAME และ NEXT_PUBLIC_CLOUDINARY_UPLOAD_PRESET ใน .env.local
        </p>
      )}
      {errorMsg && <p className="text-[10px] text-red-500">{errorMsg}</p>}
    </div>
  );
}
