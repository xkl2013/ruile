/// <reference types="vite/client" />

declare module '*.vue' {
  import { Component } from 'vue'
  const component: Component
  export default component
}

declare const __FRONTEND_VERSION__: string
declare const __FRONTEND_COMMIT__: string

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    description?: string
    navKey?: string
    minRole?: 'viewer' | 'contributor' | 'admin' | 'owner'
    requiresSystemAdmin?: boolean
    requiresTenant?: boolean
    public?: boolean
  }
}

export {}
