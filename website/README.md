# GGID.dev Marketing Website

Professional, multi-language marketing website for GGID IAM platform.

## Structure

```
website/
├── index.html          # Main page
├── assets/
│   ├── css/
│   │   └── style.css   # Full stylesheet (dark theme, responsive)
│   ├── js/
│   │   ├── i18n.js     # Internationalization (EN/ZH)
│   │   └── main.js     # Interactions, scroll reveal, tabs
│   └── svg/
│       ├── logo.svg     # GGID logo (gradient shield + key motif)
│       └── favicon.svg  # Simplified favicon
├── robots.txt
├── sitemap.xml
├── nginx.conf          # Production nginx config
└── CLOUDFLARE.md       # Cloudflare Pages deployment
```

## Features

- **Dark theme** with gradient accents (indigo/violet/cyan)
- **Multi-language**: English and Chinese, switchable in real-time
- **Responsive**: Mobile-first, breakpoints at 768px and 480px
- **Scroll animations**: Intersection Observer for reveal effects
- **Animated stat counters**
- **Code example tabs**: Go, cURL, Node.js, Python, Docker
- **SEO optimized**: meta tags, OpenGraph, sitemap, robots.txt
- **Zero dependencies**: Pure HTML/CSS/JS, no build step required

## Local Preview

```bash
cd website
python3 -m http.server 8090
# Open http://localhost:8090
```

## Deployment

### Option A: Cloudflare Pages
1. Connect GitHub repo
2. Build command: (none)
3. Output directory: `website`

### Option B: nginx
```bash
sudo cp -r website/* /var/www/ggid-website/
sudo cp website/nginx.conf /etc/nginx/sites-available/ggid.dev
sudo ln -s /etc/nginx/sites-available/ggid.dev /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

### Option C: Any static hosting
Upload the `website/` directory to any static host (S3, Netlify, Vercel, etc.)
