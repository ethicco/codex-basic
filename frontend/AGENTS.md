<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

# Frontend Guidelines

## Structure

This is a Next.js App Router application. Routes, layouts, and UI components live in `src/app/`; static assets live in `public/`; frontend configuration is in this directory.

## Commands

Run commands from `frontend/`:

```bash
npm install
npm run dev
npm run lint
npm run build
npm start
```

## Code Style and Tests

Use two-space indentation in TypeScript and TSX, follow the existing ESLint and Next.js rules, use `PascalCase` for React components, `camelCase` for variables and functions, and lowercase route folders. Prefer focused modules and CSS Modules for component styling.

Run `npm run lint` and `npm run build` after frontend changes. There is no automated test script yet; add focused tests next to new behavior, for example `page.test.tsx`.
