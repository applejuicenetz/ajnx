module.exports = {
  content: [
    "../../internal/ui/view/**/*.templ",
  ],
  theme: {
    extend: {
      colors: {
        background: "#080d18",
        surface: "#0d1320",
        border: "rgba(255,255,255,0.05)",
        primary: {
          DEFAULT: "#8fd13b",
          hover: "#a4e34a",
        },
        ink: {
          950: '#080d18',
          900: '#0d1320',
          800: '#121a2b',
          750: '#162035',
          700: '#1a2438',
          650: '#1f2a42',
          600: '#243049',
          500: '#2d3a55',
        },
        lime: {
          300: '#c6f06b',
          400: '#a4e34a',
          500: '#8fd13b',
          600: '#76b82c',
        },
        'text-main': "#F9FAFB",
        'text-muted': "#94a3b8",
        status: {
          ok: "#8fd13b",
          warn: "#FBBF24",
          err: "#EF4444",
        }
      },
      boxShadow: {
        'card': '0 20px 40px -20px rgba(0,0,0,0.5), 0 0 0 1px rgba(255,255,255,0.04) inset',
        'glow-lime': '0 0 20px rgba(143,209,59,0.25)',
      },
      keyframes: {
        pulse_node:   { '0%,100%': { opacity:'1', transform:'scale(1)' }, '50%': { opacity:'0.55', transform:'scale(1.4)' } },
        pulse_node_b: { '0%,100%': { opacity:'0.6', transform:'scale(0.9)' }, '50%': { opacity:'1', transform:'scale(1.3)' } },
        dash_flow:    { 'to': { strokeDashoffset:'-40' } },
        ekg:          { '0%': { strokeDashoffset:'120' }, '100%': { strokeDashoffset:'0' } },
        blink:        { '0%,100%': { opacity:'1' }, '50%': { opacity:'0.3' } },
      },
      animation: {
        'pulse-node':   'pulse_node 2.6s ease-in-out infinite',
        'pulse-node-b': 'pulse_node_b 3.2s ease-in-out infinite',
        'dash-flow':    'dash_flow 3s linear infinite',
        'ekg':          'ekg 2.4s linear infinite',
        'blink':        'blink 1.6s ease-in-out infinite',
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
  plugins: [],
}
