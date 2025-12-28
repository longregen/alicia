/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'media',
  theme: {
    extend: {
      colors: {
        // Alicia Audio Style Guide Colors
        // Background Colors
        'main-bg': '#050e24',
        'container-bg': 'rgba(10, 20, 50, 0.7)',
        'surface-bg': 'rgba(20, 40, 90, 0.6)',
        'message-received-bg': 'rgba(32, 124, 229, 0.2)',
        'message-sent-bg': '#011327cc',
        
        // Brand Colors
        'primary-blue': 'rgba(32, 124, 229, 0.8)',
        'primary-blue-hover': 'rgba(32, 124, 229, 0.9)',
        'primary-blue-active': 'rgba(32, 125, 229, 1)',
        'primary-blue-glow': 'rgba(32, 124, 229, 0.6)',
        
        // State Colors
        'active-speaking': 'rgba(40, 200, 80, 0.9)',
        'inactive-disabled': 'rgba(128, 128, 128, 0.8)',
        
        // Text Colors
        'primary-text': '#e0e6ff',
        'white-text': 'white',
        'muted-text': 'rgba(255, 255, 255, 0.6)',
        
        // Special Colors
        'reasoning': '#9c27b0',
        'tool-use': '#2196f3',
        'tool-result': '#ff9800',
        'translation-complete': '#4CAF50',
        
        // Keep existing Tailwind-style color scales for components that need them
        alicia: {
          50: '#f0f9ff',
          100: '#e0f2fe',
          200: '#bae6fd',
          300: '#7dd3fc',
          400: '#38bdf8',
          500: '#207ce5', // Primary blue
          600: '#207ce5e6', // Primary blue hover
          700: '#207de5', // Primary blue active
          800: '#075985',
          900: '#0c4a6e',
          950: '#082f49',
        },
        success: {
          50: '#f0fdf4',
          100: '#dcfce7',
          200: '#bbf7d0',
          300: '#86efac',
          400: '#4ade80',
          500: '#28c850', // Active/Speaking green
          600: '#16a34a',
          700: '#15803d',
          800: '#166534',
          900: '#14532d',
          950: '#052e16',
        },
        error: {
          50: '#fef2f2',
          100: '#fee2e2',
          200: '#fecaca',
          300: '#fca5a5',
          400: '#f87171',
          500: '#ff6464', // Error red
          600: '#dc2626',
          700: '#b91c1c',
          800: '#991b1b',
          900: '#7f1d1d',
          950: '#450a0a',
        },
        surface: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          600: '#475569',
          700: '#334155',
          800: '#1e293b',
          900: '#0f172a',
          950: '#020617',
        }
      },
      animation: {
        'pulse-slow': 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'bounce-gentle': 'bounce 1s infinite',
        'fade-in': 'fadeIn 0.3s ease-in-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'wave': 'wave 2s ease-in-out infinite',
        'shimmer': 'shimmer 1.5s infinite',
        'recording-pulse': 'recordingPulse 1.5s ease-in-out infinite',
        'volume-bar': 'volumeBar 0.8s ease-in-out infinite',
        'ping': 'ping 1s cubic-bezier(0, 0, 0.2, 1) infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        wave: {
          '0%, 100%': { transform: 'scaleY(0.5)' },
          '50%': { transform: 'scaleY(1)' },
        },
        shimmer: {
          '0%': { transform: 'translateX(-100%)' },
          '100%': { transform: 'translateX(100%)' },
        },
        recordingPulse: {
          '0%, 100%': { 
            opacity: '1',
            transform: 'scale(1)',
          },
          '50%': { 
            opacity: '0.8',
            transform: 'scale(1.05)',
          },
        },
        volumeBar: {
          '0%': { height: '20%' },
          '50%': { height: '100%' },
          '100%': { height: '20%' },
        },
        ping: {
          '75%, 100%': {
            transform: 'scale(2)',
            opacity: '0',
          },
        },
      },
    },
  },
  plugins: [
    function({ addUtilities }) {
      addUtilities({
        '.animation-duration-2000': {
          'animation-duration': '2000ms',
        },
      })
    },
  ],
}