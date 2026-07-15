/* ============================================
   GGID.dev — Main JavaScript
   Interactions, animations, UI behavior
   ============================================ */

document.addEventListener('DOMContentLoaded', () => {
  // ---- Navigation scroll effect ----
  const nav = document.querySelector('nav');
  let lastScroll = 0;

  window.addEventListener('scroll', () => {
    const scroll = window.scrollY;
    nav.classList.toggle('scrolled', scroll > 20);
    lastScroll = scroll;
  });

  // ---- Mobile menu ----
  const mobileToggle = document.querySelector('.mobile-toggle');
  const navLinks = document.querySelector('.nav-links');

  if (mobileToggle) {
    mobileToggle.addEventListener('click', () => {
      navLinks.classList.toggle('mobile-open');
    });
  }

  // Close mobile menu on link click
  navLinks?.querySelectorAll('a').forEach(link => {
    link.addEventListener('click', () => {
      navLinks.classList.remove('mobile-open');
    });
  });

  // ---- Language switcher ----
  const langBtn = document.querySelector('.lang-btn');
  const langDropdown = document.querySelector('.lang-dropdown');

  if (langBtn) {
    langBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      langDropdown.classList.toggle('open');
    });

    document.addEventListener('click', (e) => {
      if (!langDropdown.contains(e.target) && !langBtn.contains(e.target)) {
        langDropdown.classList.remove('open');
      }
    });
  }

  document.querySelectorAll('.lang-dropdown button').forEach(btn => {
    btn.addEventListener('click', () => {
      const lang = btn.dataset.lang;
      if (typeof applyTranslations === 'function') {
        applyTranslations(lang);
      }
      langDropdown.classList.remove('open');
    });
  });

  // ---- Code tabs ----
  const codeTabs = document.querySelectorAll('.code-tab');
  const codeBlocks = document.querySelectorAll('.code-block');

  codeTabs.forEach(tab => {
    tab.addEventListener('click', () => {
      const target = tab.dataset.tab;

      codeTabs.forEach(t => t.classList.remove('active'));
      codeBlocks.forEach(b => b.classList.remove('active'));

      tab.classList.add('active');
      document.getElementById(`code-${target}`)?.classList.add('active');
    });
  });

  // ---- Scroll reveal ----
  const revealElements = document.querySelectorAll('.reveal');

  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        entry.target.classList.add('visible');
        observer.unobserve(entry.target);
      }
    });
  }, {
    threshold: 0.1,
    rootMargin: '0px 0px -50px 0px'
  });

  revealElements.forEach(el => observer.observe(el));

  // ---- Animated stat counters ----
  const stats = document.querySelectorAll('.stat-num[data-count]');
  let statsAnimated = false;

  const animateStats = () => {
    if (statsAnimated) return;
    const statsBar = document.querySelector('.stats-bar');
    if (!statsBar) return;

    const rect = statsBar.getBoundingClientRect();
    if (rect.top < window.innerHeight && rect.bottom > 0) {
      statsAnimated = true;
      stats.forEach(stat => {
        const target = parseInt(stat.dataset.count);
        const suffix = stat.dataset.suffix || '';
        const duration = 1500;
        const startTime = performance.now();

        const update = (currentTime) => {
          const elapsed = currentTime - startTime;
          const progress = Math.min(elapsed / duration, 1);
          const eased = 1 - Math.pow(1 - progress, 3);
          const value = Math.floor(target * eased);
          stat.textContent = value + suffix;
          if (progress < 1) requestAnimationFrame(update);
        };

        requestAnimationFrame(update);
      });
    }
  };

  window.addEventListener('scroll', animateStats);
  animateStats();

  // ---- Smooth scroll for anchor links ----
  document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', (e) => {
      const href = anchor.getAttribute('href');
      if (href === '#') return;

      const target = document.querySelector(href);
      if (target) {
        e.preventDefault();
        const navHeight = parseInt(getComputedStyle(document.documentElement)
          .getPropertyValue('--nav-height'));
        const top = target.getBoundingClientRect().top + window.scrollY - navHeight - 20;
        window.scrollTo({ top, behavior: 'smooth' });
      }
    });
  });

  // ---- Terminal typing effect ----
  const terminal = document.querySelector('.terminal-body');
  if (terminal && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    // Subtle glow animation already handled by CSS
  }
});
