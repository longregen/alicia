import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'
import tailwindcss from '@tailwindcss/vite'

// Versions from package.json for cache-busting
const vendorVersions: Record<string, string> = {
  'react-dom': '19.2.3',
  'react': '19.2.3',
  'livekit-client': '2.16.1',
  '@livekit/components-react': '2.9.17',
  'msgpackr': '1.11.8',
};

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    {
      name: 'serve-sql-wasm',
      configureServer(server) {
        server.middlewares.use('/sql-wasm.wasm', (_req, res) => {
          const wasmPath = resolve(__dirname, 'node_modules/sql.js/dist/sql-wasm.wasm')
          res.setHeader('Content-Type', 'application/wasm')
          import('fs').then(fs => {
            fs.createReadStream(wasmPath).pipe(res)
          })
        })
      },
    },
    {
      name: 'cache-vad-assets',
      configureServer(server) {
        // Add proper MIME types and long-term caching for VAD/ONNX assets
        server.middlewares.use((req, res, next) => {
          const url = req.url || '';

          // Match VAD and ONNX static assets
          if (url.startsWith('/js/lib/') || url.startsWith('/onnx/') || url.startsWith('/models/')) {
            // Set proper MIME types
            if (url.endsWith('.js')) {
              res.setHeader('Content-Type', 'application/javascript');
            } else if (url.endsWith('.wasm')) {
              res.setHeader('Content-Type', 'application/wasm');
            } else if (url.endsWith('.onnx')) {
              res.setHeader('Content-Type', 'application/octet-stream');
            }

            // Cache for 1 year (immutable versioned assets)
            res.setHeader('Cache-Control', 'public, max-age=31536000, immutable');
          }

          next();
        });
      },
      configurePreviewServer(server) {
        // Same middleware for preview server
        server.middlewares.use((req, res, next) => {
          const url = req.url || '';

          if (url.startsWith('/js/lib/') || url.startsWith('/onnx/') || url.startsWith('/models/')) {
            if (url.endsWith('.js')) {
              res.setHeader('Content-Type', 'application/javascript');
            } else if (url.endsWith('.wasm')) {
              res.setHeader('Content-Type', 'application/wasm');
            } else if (url.endsWith('.onnx')) {
              res.setHeader('Content-Type', 'application/octet-stream');
            }

            res.setHeader('Cache-Control', 'public, max-age=31536000, immutable');
          }

          next();
        });
      },
    },
  ],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            // Extract package name from path
            const match = id.match(/node_modules\/(@[^/]+\/[^/]+|[^/]+)/);
            if (match) {
              const pkg = match[1];
              // Group react-dom with react
              if (pkg === 'react-dom' || pkg === 'react' || pkg === 'scheduler') {
                return `react-${vendorVersions['react']}`;
              }
              // Group livekit packages
              if (pkg.includes('livekit') || pkg.includes('pion') || pkg.includes('sdp')) {
                return `livekit-${vendorVersions['livekit-client']}`;
              }
              // Named packages with versions
              if (vendorVersions[pkg]) {
                return `${pkg.replace('/', '-')}-${vendorVersions[pkg]}`;
              }
              // Other dependencies go to a common vendor chunk
              return 'vendor';
            }
          }
        },
      },
    },
  },
  server: {
    port: 3001,
    allowedHosts: ['alicia.hjkl.lol'],
    // Disable proxy during e2e tests so Playwright can intercept API calls
    proxy: process.env.PLAYWRIGHT_TEST ? undefined : {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:8090',
        changeOrigin: true,
      }
    },
  },
  test: {
    watch: false,
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    exclude: [
      '**/node_modules/**',
      '**/dist/**',
      '**/e2e/**',
    ],
    environmentOptions: {
      jsdom: {
        resources: 'usable',
      },
    },
    env: {
      NODE_ENV: 'development',
    },
    pool: 'threads',
    maxWorkers: 4,
    minWorkers: 1,
    isolate: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/mockData',
        'dist/',
      ]
    }
  }
})
