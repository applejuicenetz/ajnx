/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "../../internal/ui/view/**/*.templ",
  ],
  theme: {
    extend: {
      colors: {
        background: "#111827",
        surface: "#1F2937",
        'surface-hover': "#2D3748",
        border: "#374151",
        primary: {
          DEFAULT: "#7CB342",
        },
        'text-main': "#F9FAFB",
        'text-muted': "#D1D5DB",
        status: {
          ok: "#7CB342",
          warn: "#FBBF24",
          err: "#EF4444",
        }
      },
      fontFamily: {
        sans: ['ui-sans-serif', 'system-ui', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'Roboto', '"Helvetica Neue"', 'Arial', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
