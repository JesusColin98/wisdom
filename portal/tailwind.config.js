/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        nexus: {
          dark: "#0a0a0a",
          gold: "#d4af37",
          blue: "#00b4d8",
          purple: "#7b2cbf"
        }
      }
    },
  },
  plugins: [],
}
