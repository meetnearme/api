/* global module */
module.exports = {
  plugins: ['html'],
  extends: ['eslint:recommended'],
  env: {
    browser: true,
    es6: true
  },
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module'
  },
  settings: {
    'html/html-extensions': ['.html', '.templ'],
    'html/indent': 'tab+',
    'html/report-bad-indent': 'warn',
    'html/javascript-mime-types': ['text/javascript']
  },
  rules: {
    'no-unused-vars': ['warn', {
      // All Alpine x-data functions must begin with
      // `get` and end with `State`, if so we skip linting them
      'varsIgnorePattern': '^get(.*)State$'
    }],
    'no-undef': 'error',
    'no-console': 'warn',
    'space-infix-ops': 'error',
    'no-whitespace-before-property': 'error'  // This will catch the space before .querySelector
  },
  globals: {
    // Alpine.js globals
    'Alpine': 'readonly',
    '$data': 'readonly',
    '$store': 'readonly',
    '$refs': 'readonly',
    '$el': 'readonly'
  }
}
