/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["internal/server/templates/*.templ"],
  theme: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/typography'),
    require('daisyui'),
  ],
}
