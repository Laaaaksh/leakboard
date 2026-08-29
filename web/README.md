# Leakboard dashboard frontend

React + TypeScript + Vite. This builds the dashboard UI that `internal/webui` embeds into the
`leakboard` Go binary — see the root [README](../README.md) for what the product does, and
[../CONTRIBUTING.md](../CONTRIBUTING.md) for the required "rebuild and commit
`internal/webui/dist`" step after any change here.

```bash
npm install
npm run dev     # dev server with API requests proxied to :8080 (see vite.config.ts)
npm run build   # production build, output in dist/
npm run lint    # oxlint
```
