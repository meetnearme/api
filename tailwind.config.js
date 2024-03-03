/** @type {import('tailwindcss').Config} */

export default {
  content: ['functions/lambda/views/**/*.templ'],
  theme: {
    fontSize: {
      sm: '0.8rem',
      base: '1rem',
      xl: '1.25rem',
      '2xl': '1.563rem',
      '3xl': '1.953rem',
      '4xl': '2.441rem',
      '5xl': '3.052rem',
    },
    extend: {},
  },
  plugins: [require('daisyui'), require('@tailwindcss/typography')],
  daisyui: {
    themes: [
      {
        cyberpunk: {
          ...require('daisyui/src/theming/themes')['cyberpunk'],
          primary: '#00c9fc',
          secondary: '#c64000',
          accent: '#008e3a',
          neutral: '#070508',
          'base-100': '#fffff1',
          info: '#00ecff',
          success: '#b1e240',
          warning: '#ff8800',
          error: '#ce0040',
        },
      },
    ],
  },
};
