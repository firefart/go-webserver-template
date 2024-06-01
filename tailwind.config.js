/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["internal/server/templates/*.templ"],
  theme: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
    require('daisyui'),
  ],
}
