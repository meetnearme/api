/** @type {import('tailwindcss').Config} */
export default {
  content: ['app/templates/**/*.templ'],
  theme: {
    extend: {},
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: ['cupcake', 'luxury'],
  },
};