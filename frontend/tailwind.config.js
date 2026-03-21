/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,jsx}",
  ],
  darkMode: 'media', // Automatically use system preference
  theme: {
    extend: {
      colors: {
        // Brand Colors
        brand: {
          primary: '#fb6205',      // Vibrant Orange
          dark: '#0a1527',         // Deep Navy Blue
          gray: '#888d95',         // Neutral Gray
          'primary-light': '#ff8c3a',
          'primary-dark': '#d94a00',
          'dark-light': '#1a2540',
          'dark-lighter': '#2a3550',
          'gray-light': '#a8adb5',
          'gray-dark': '#686d75',
        },
        // Legacy primary colors (kept for compatibility)
        primary: {
          50: '#f7f7f7',
          100: '#e3e3e3',
          200: '#c8c8c8',
          300: '#a4a4a4',
          400: '#818181',
          500: '#666666',
          600: '#515151',
          700: '#434343',
          800: '#383838',
          900: '#1a1a1a',
        }
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      }
    },
  },
  plugins: [],
}
