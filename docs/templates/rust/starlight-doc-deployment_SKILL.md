# Skill: Starlight Documentation Deployment
**Domain:** Astro / GitHub Actions
**Scope:** Project presentation website generation and CI/CD routing.

## Execution Parameters
* Initialize Astro project using the Starlight template within the `/docs` directory.
* Configure `pages.yml` in `.github/workflows/` to trigger on pushes to the `main` branch.
* Ensure CI/CD workflow provisions Node.js environment, installs dependencies via `npm ci`, executes Astro build, and uploads artifacts to GitHub Pages deployment pipeline.