import { defineConfig, globalIgnores } from 'eslint/config'
import nextVitals from 'eslint-config-next/core-web-vitals'
import nextTs from 'eslint-config-next/typescript'

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  {
    settings: {
      react: { version: '19.2' },   // ← ปิดการ auto-detect
    },
    rules: {
      'react-hooks/set-state-in-effect': 'warn',     // TODO: กลับเป็น error หลัง refactor

      // ({ node, ...props }) คือวิธีมาตรฐานในการ "โยนทิ้ง" prop ก่อน spread ลง DOM
      // eslint-config-next ตั้ง rule นี้เป็น 'warn' เปล่าๆ ซึ่ง reset option กลับเป็นค่า default
      // ของ ESLint core (ignoreRestSiblings: false) เลยฟ้อง node ที่เราตั้งใจตัดทิ้ง
      '@typescript-eslint/no-unused-vars': ['warn', { ignoreRestSiblings: true }],
    },
  },
  globalIgnores(['.next/**', 'out/**', 'build/**', 'next-env.d.ts']),
])

export default eslintConfig