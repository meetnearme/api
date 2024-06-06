/** @type {import('tailwindcss').Config} */

export default {
  content: ['**/*.templ'],
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
    darkTheme: 'meetnearme',
    themes: [
      {
        meetnearme: {
          ...require('daisyui/src/theming/themes')['cupcake'],
          'color-scheme': 'light',
          fontFamily:
            'ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,Liberation Mono,Courier New,monospace',
          '--rounded-box': '1rem', // border radius rounded-box utility class, used in card and other large boxes
          '--rounded-btn': '0.5rem', // border radius rounded-btn utility class, used in buttons and similar element
          '--rounded-badge': '1.9rem',
          '--tab-radius': '0.5rem',
          // '--rounded-box': '0',
          // '--rounded-btn': '0',
          // '--rounded-badge': '0',
          // '--tab-radius': '0',
          primary: '#00ceff',
          secondary: '#5eead4',
          accent: '#f0abfc',
          neutral: '#190c04',
          'neutral-content': '#00ceff',
          'base-100': '#fffbe6',
          info: '#9fb9f9',
          success: '#74ea62',
          warning: '#ffc458',
          error: '#ff7f7f',
          primary: '#00ceff',
          secondary: '#5eead4',
          accent: '#f0abfc',
          neutral: '#190c04',
        },
      },
    ],
  },
};
