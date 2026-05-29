/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{js,ts}'],
  theme: {
    extend: {
      colors: {
        gray: {
          950: '#0a0f1a',
        }
      }
    },
  },
  plugins: [],
}
