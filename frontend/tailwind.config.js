/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      fontSize: {
        'xxs': ['10px', { lineHeight: '1.4' }],
        'xs-body': ['11px', { lineHeight: '1.5' }],
        'sm-body': ['13px', { lineHeight: '1.6' }],
      },
      spacing: {
        'compact': '0.5rem',
        'standard': '1rem',
        'spacious': '1.5rem',
      },
    },
  },
};
