/** @type {import('tailwindcss').Config} */

export default {
  mode: 'jit',
  purge: ['**/*.templ'],
  content: ['**/*.templ'],
  theme: {
    extend: {
      aspectRatio: {
        '4/1': '4 / 1',
      },
    },
    fontSize: {
      sm: '0.8rem',
      base: '1rem',
      lg: '1.15rem',
      xl: '1.25rem',
      '2xl': '1.563rem',
      '3xl': '1.953rem',
      '4xl': '2.441rem',
      '5xl': '3.052rem',
    },
    container: {
      padding: {
        DEFAULT: '1rem',
        sm: '2rem',
        md: '3rem',
        // lg: '8rem',
        // xl: '10rem',
        // '2xl': '12rem',
      },
    },
  },
  corePlugins: {
    container: false,
  },
  plugins: [
    require('daisyui'),
    require('@tailwindcss/typography'),
    ({ addComponents }) => {
      addComponents({
        '.alert': {
          gridAutoFlow: 'column',
        },
        '.container': {
          maxWidth: '100%',
          width: '100%',
          '@screen sm': {
            width: '100%',
          },
          '@screen md': {
            width: '100%',
          },
          '@screen lg': {
            width: '960px',
          },
        },
        '.select-bordered': {
          borderColor: 'var(--fallback-bc,oklch(var(--bc)/0.6))'
        },
        '.input-bordered': {
          borderColor: 'var(--fallback-bc,oklch(var(--bc)/0.6))'
        },
        '.textarea-bordered': {
          borderColor: 'var(--fallback-bc,oklch(var(--bc)/0.6))'
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
        '.tab:is(input[type="radio"])': {
          borderBottomRightRadius: 'inherit',
          borderBottomLeftRadius: 'inherit',
        },
        '.carousel-control-left': {
          display: 'none',
          '@screen md': {
            display: 'block',
          },
        },
        '.carousel-control-right': {
          display: 'none',
          '@screen md': {
            display: 'block',
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
          '--rounded-box': '0.5rem',
          '--rounded-btn': '0.25rem',
          '--rounded-badge': '1rem',
          '--tab-radius': '0.25rem',
          '--btn-bg-inverted': '100% 0 0', // White in OKLCH
          '--btn-bg-inverted-content': '0% 0 0', // Black in OKLCH
          primary: '#39FF14',
          'primary-content': '#011600',
          secondary: '#FF4500',
          'secondary-content': '#eeeeee',
          accent: '#FF69B4',
          'accent-content': '#16040c',
          neutral: '#cccccc',
          'neutral-content': '#000000',
          'base-100': '#000000',
          'base-200': '#2a2a2a',
          'base-300': '#454545',
          'base-content': '#F5F5F5',
          info: '#ffa914',
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
