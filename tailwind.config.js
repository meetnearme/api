/** @type {import('tailwindcss').Config} */

const sharedTheme = {
  fontFamily:
    'Ubuntu Mono,ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,Liberation Mono,Courier New,monospace',
  '--rounded-box': '0.5rem',
  '--rounded-btn': '0.25rem',
  '--rounded-badge': '1rem',
  '--tab-radius': '0.25rem',
  info: '#84ccff',
  'info-content': '#000000',
  success: '#74ea62',
  'success-content': '#000000',
  warning: '#ffc458',
  'warning-content': '#000000',
  error: '#AE1335',
  'error-content': '#FFFFFF',
};

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
    // eslint-disable-next-line no-undef
    require('daisyui'),
    // eslint-disable-next-line no-undef
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
          borderColor: 'var(--fallback-bc,oklch(var(--bc)/0.6))',
        },
        '.input-bordered': {
          borderColor: 'var(--fallback-bc,oklch(var(--bc)/0.6))',
        },
        '.textarea-bordered': {
          borderColor: 'var(--fallback-bc,oklch(var(--bc)/0.6))',
        },
        '.main-bg': {
          width: '100vw',
          position: 'fixed',
          left: '0',
          top: '40vw',
          opacity: '0.15',
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
        '.progress.input-bottom': {
          borderRadius: '0',
          height: '0.25rem',
        },
        '.btn': {
          borderRadius: '4rem',
        },
        '.checkbox': {
          borderWidth: '2px',
        },
      });
    },
  ],
  daisyui: {
    defaultTheme: 'meetnearmelight',
    darkTheme: 'meetnearmedark',
    lightTheme: 'meetnearmelight',
    themes: [
      {
        // Light theme
        meetnearmelight: {
          'color-scheme': 'light',
          primary: 'hsl(239, 84%, 67%)',
          'primary-content': '#ffffff',
          secondary: 'hsl(239, 84%, 72%)',
          'secondary-content': '#ffffff',
          accent: 'hsl(239, 84%, 67%)', // not currently used
          'accent-content': '#ffffff',
          'base-100': '#ffffff',
          'base-200': '#f0f0f0',
          'base-300': '#e1e1e1',
          'base-content': '#202020',
          '--btn-bg-inverted': '100% 0 0', // White in OKLCH
          '--btn-bg-inverted-content': '0% 0 0', // Black in OKLCH
          ...sharedTheme,
        },
        // Dark theme
        meetnearmedark: {
          'color-scheme': 'dark',
          primary: 'hsl(239, 84%, 67%)',
          'primary-content': '#ffffff',
          secondary: 'hsl(235, 100%, 86%)',
          'secondary-content': 'hsl(240, 10%, 8%)',
          accent: 'hsl(239, 84%, 80%)', // not currently used
          'accent-content': '#000000',
          'base-100': 'hsl(240, 10%, 8%)',
          'base-200': 'hsl(240, 8%, 12%)',
          'base-300': '#48414e',
          'base-content': '#F5F5F5',
          '--btn-bg-inverted': '0% 0 0', // Black in OKLCH
          '--btn-bg-inverted-content': '100% 0 0', // White in OKLCH
          ...sharedTheme,
        },
      },
    ],
  },
};
