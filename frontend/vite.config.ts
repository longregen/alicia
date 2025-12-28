import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

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
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  },
  test: {
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
