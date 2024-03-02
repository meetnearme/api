/** @type {import('tailwindcss').Config} */

export default {
  content: ['app/templates/**/*.templ'],
  theme: {
    extend: {},
  },
  plugins: [require("@tailwindcss/typography"), require("daisyui")]
  daisyui: {
    themes: ['cyberpunk'],
  },
};
