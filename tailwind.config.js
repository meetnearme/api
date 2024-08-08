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
          '&.top': {
            top: '0',
          },
        },
        '.header-hero': {
          maxWidth: '70vw',
          margin: '0 auto',
          '@screen sm': {
            maxWidth: '90vw',
          },
          '@screen md': {
            maxWidth: '90vw',
          },
          '@screen lg': {
            maxWidth: '70vw',
          },
          '@screen xl': {
            maxWidth: '70vw',
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
          ...require('daisyui/src/theming/themes')['dark'],
          'color-scheme': 'dark',
          fontFamily:
            'Ubuntu Mono,ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,Liberation Mono,Courier New,monospace',
          // '--rounded-box': '1rem', // border radius rounded-box utility class, used in card and other large boxes
          // '--rounded-btn': '0.5rem', // border radius rounded-btn utility class, used in buttons and similar element
          // '--rounded-badge': '1.9rem',
          // '--tab-radius': '0.5rem',
          '--rounded-box': '0',
          '--rounded-btn': '0',
          '--rounded-badge': '0',
          '--tab-radius': '0',
          primary: '#39FF14',
          'primary-content': '#011600',
          secondary: '#FF4500',
          'secondary-content': '#160200',
          accent: '#FF69B4',
          'accent-content': '#16040c',
          neutral: '#a8a29e',
          'neutral-content': '#000000',
          'base-100': '#000000',
          'base-200': '#000000',
          'base-300': '#000000',
          'base-content': '#bebebe',
          info: '#7cbbee',
          'info-content': '#000000',
          success: '#74ea62',
          'success-content': '#000000',
          warning: '#ffc458',
          'warning-content': '#000000',
          error: '#e11d48',
          'error-content': '#000000',
        },
      },
    ],
  },
};
