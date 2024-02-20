/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./components/*.{templ,go}"],
  theme: {
    extend: {
      typography: {
        DEFAULT: {
          css: {
            maxWidth: '100ch',
          }
        }
      }
    }
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}

