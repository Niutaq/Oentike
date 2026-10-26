# Contributing to Oentike

Thank you for your interest in contributing.

## Getting Started

1. Fork the repository and clone locally.
2. Run `mise trust && mise install` to pin tool versions.
3. Run `./oentike doctor` to verify your environment.
4. Run `./oentike setup` to install dependencies.
5. Run `./oentike check` to verify everything passes before submitting changes.

## Code Style

- **Go**: formatted with `gofumpt` (CI enforces this).
- **TypeScript/Astro**: follow existing patterns in `oentike-web/src/`.
- **Comments**: explain *why*, not *what*. No filler comments.

## Atlas Contributions

- All species cards must conform to `oentike-atlas/schema/card.schema.json`.
- Supply four anatomical SVGs per species (cap, stem, section, hymenophore).
- Cite sources. Do not copy text or images from external atlases.
- Run `./oentike atlas` to validate before submitting.

## Pull Requests

- One logical change per PR.
- Include a short description of *what* and *why*.
- CI must pass (`backend-checks` + `frontend-checks`).
