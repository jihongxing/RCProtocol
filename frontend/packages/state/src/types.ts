export interface User {
  user_id: string
  email: string
  display_name: string
  org_id: string
  org_name: string
  role: string
  brand_id?: string
}

export interface OrgContext {
  org_id: string
  org_name: string
  org_type: string
}

export interface BrandContext {
  brand_id: string
  brand_name: string
}

export interface StorageAdapter {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}
