import { type ReactNode } from 'react'

interface FieldProps {
  label: string
  required?: boolean
  children: ReactNode
  className?: string
}

// 表单字段容器：统一 label + 控件排版，减少重复样板
export default function Field({ label, required, children, className = '' }: FieldProps) {
  return (
    <label className={`form-control w-full ${className}`}>
      <div className="label">
        <span className="label-text">
          {label}
          {required && <span className="text-error"> *</span>}
        </span>
      </div>
      {children}
    </label>
  )
}
