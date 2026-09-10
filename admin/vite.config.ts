import { fileURLToPath, URL } from 'node:url'
import { resolve, dirname } from 'node:path'
import { existsSync, readFileSync } from 'node:fs'
import { execSync } from 'node:child_process'
import { createRequire } from 'node:module'
import { defineConfig } from 'vite'
import type { Plugin as VitePlugin } from 'vite'
import type { Plugin as EsbuildPlugin } from 'esbuild'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'

const __dirname = dirname(fileURLToPath(import.meta.url))
const require = createRequire(import.meta.url)

const pkg = require('./package.json') as { version?: string }
const FRONTEND_VERSION = pkg.version ?? 'unknown'
const DEV_PROXY_TARGET =
  process.env.VITE_DEV_PROXY_TARGET ||
  process.env.FRONTEND_BACKEND_URL ||
  'http://localhost:8080'
const VUE_ROUTER_DEVTOOLS_ASSIGN = 'instance.__vrv_devtools = info;'

function resolveFrontendCommit(): string {
  const fromEnv = process.env.VITE_FRONTEND_COMMIT || process.env.GITHUB_SHA
  if (fromEnv) return fromEnv.slice(0, 7)
  try {
    return execSync('git rev-parse --short HEAD', { stdio: ['ignore', 'pipe', 'ignore'] })
      .toString()
      .trim()
  } catch {
    return 'unknown'
  }
}

function resolveVueOfficePptxEntry(): string {
  try {
    const pkgDir = dirname(require.resolve('@vue-office/pptx/package.json'))
    const candidates = [
      resolve(pkgDir, 'lib/v3/index.js'),
      resolve(pkgDir, 'lib/index.js'),
      resolve(pkgDir, 'lib/v3/vue-office-pptx.mjs'),
    ]
    return candidates.find((candidate) => existsSync(candidate)) ?? '@vue-office/pptx'
  } catch {
    return '@vue-office/pptx'
  }
}

function patchVueRouterDevtoolsNullRef(code: string): string {
  if (
    !code.includes(VUE_ROUTER_DEVTOOLS_ASSIGN) ||
    code.includes(`if (instance) ${VUE_ROUTER_DEVTOOLS_ASSIGN}`)
  ) {
    return code
  }

  return code.replace(
    VUE_ROUTER_DEVTOOLS_ASSIGN,
    `if (instance) ${VUE_ROUTER_DEVTOOLS_ASSIGN}`,
  )
}

function vueRouterDevtoolsNullRefPatch(): VitePlugin {
  const vueRouterDistFilter = /vue-router[/\\]dist[/\\]vue-router\.mjs$/
  const optimizedVueRouterFilter = /[/\\]node_modules[/\\]\.vite[/\\]deps[/\\]vue-router\.js(?:\?.*)?$/
  const esbuildPatch: EsbuildPlugin = {
    name: 'admin-vue-router-devtools-null-ref-patch',
    setup(build) {
      build.onLoad({ filter: vueRouterDistFilter }, (args) => ({
        contents: patchVueRouterDevtoolsNullRef(readFileSync(args.path, 'utf8')),
        loader: 'js',
      }))
    },
  }

  return {
    name: 'admin-vue-router-devtools-null-ref-patch',
    enforce: 'pre',
    config() {
      return {
        optimizeDeps: {
          esbuildOptions: {
            plugins: [esbuildPatch],
          },
        },
      }
    },
    transform(code, id) {
      if (!vueRouterDistFilter.test(id) && !optimizedVueRouterFilter.test(id)) {
        return null
      }

      const patched = patchVueRouterDevtoolsNullRef(code)
      return patched === code ? null : { code: patched, map: null }
    },
  }
}

export default defineConfig({
  base: process.env.VITE_ADMIN_BASE || '/admin/',
  publicDir: fileURLToPath(new URL('../frontend/public', import.meta.url)),
  define: {
    __FRONTEND_VERSION__: JSON.stringify(FRONTEND_VERSION),
    __FRONTEND_COMMIT__: JSON.stringify(resolveFrontendCommit()),
    'import.meta.env.VITE_API_BASE_URL': JSON.stringify(process.env.VITE_ADMIN_API_BASE_URL || '/'),
  },
  plugins: [
    vueRouterDevtoolsNullRefPatch(),
    vue(),
    vueJsx(),
  ],
  resolve: {
    // Admin reuses frontend/src, so keep one Vue/Pinia/router runtime.
    dedupe: ['vue', 'pinia', 'vue-router'],
    alias: {
      '@': fileURLToPath(new URL('../frontend/src', import.meta.url)),
      '@admin': fileURLToPath(new URL('./src', import.meta.url)),
      '@vue-office/pptx': resolveVueOfficePptxEntry(),
    },
  },
  server: {
    port: 8082,
    host: true,
    proxy: {
      '/api': {
        target: DEV_PROXY_TARGET,
        changeOrigin: true,
        secure: false,
      },
      '/files': {
        target: DEV_PROXY_TARGET,
        changeOrigin: true,
        secure: false,
      },
    },
  },
})
