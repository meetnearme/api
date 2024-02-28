/** @type {import('tailwindcss').Config} */
export default {
  content: ['templates/**/*.templ'],
  theme: {
    extend: {},
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: ['cupcake', 'luxury'],
  },
};