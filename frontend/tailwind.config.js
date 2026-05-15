export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        card: 'hsl(var(--card))',
        border: 'hsl(var(--border))',
        primary: 'hsl(var(--primary))',
        secondary: 'hsl(var(--secondary))',
        muted: 'hsl(var(--muted))',
        accent: 'hsl(var(--accent))',
        success: 'hsl(var(--success))',
        danger: 'hsl(var(--danger))',
        ink: '#17202a',
        field: 'hsl(var(--background))',
        line: 'hsl(var(--border))',
        warn: '#b45309',
      },
      boxShadow: {
        panel: '0 24px 70px -42px rgb(15 23 42 / 0.55)',
      },
    },
  },
  plugins: [],
};
