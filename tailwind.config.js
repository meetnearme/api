/** @type {import('tailwindcss').Config} */

export default {
  content: ['functions/gateway/templates/**/*.templ'],
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
    container: {
      padding: {
        DEFAULT: '1rem',
        // sm: '2rem',
        // md: '3rem',
        // lg: '8rem',
        // xl: '10rem',
        // '2xl': '12rem',
      },
    },
  },
  plugins: [
    require('daisyui'),
    require('@tailwindcss/typography'),
    ({ addComponents }) => {
      addComponents({
        '.container': {
          maxWidth: '100%',
          '@screen sm': {
            maxWidth: '95vw',
          },
          '@screen md': {
            maxWidth: '95vw',
          },
          '@screen lg': {
            maxWidth: '1280px',
          },
          '@screen xl': {
            maxWidth: '1400px',
          },
        },
        '.main-bg': {
          width: '100vw',
          position: 'fixed',
          left: '0',
          top: '40vw',
          opacity: '0.25',
          zIndex: '-1',
          '@screen sm': {
            top: '50vw',
          },
          '@screen md': {
            top: '45vw',
          },
          '@screen lg': {
            top: '20vw',
          },
          '@screen xl': {
            top: '20vw',
          },
          // '@media (max-aspect-ratio: 1/1)': {
          //   top: '80vh',
          // },
        },
        '.header-hero': {
          minHeight: '50vw',
          maxWidth: '70vw',
          margin: '0 auto',
          '@screen sm': {
            maxWidth: '90vw',
          },
          '@screen md': {
            maxWidth: '90vw',
            minHeight: '30vw',
          },
          '@screen lg': {
            maxWidth: '70vw',
            minHeight: '20vw',
          },
          '@screen xl': {
            maxWidth: '70vw',
            minHeight: '20vw',
          },
        },
      });
    },
  ],
  daisyui: {
    darkTheme: 'cyberpunk',
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
