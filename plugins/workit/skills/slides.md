# slides.md

Google Slides: create/copy/export decks, slide CRUD, notes, and markdown-based generation.

## Deck operations
- `wk slides create <title>`
- `wk slides create-from-markdown <title> --content-file slides.md`
- `wk slides info <presentationId>`
- `wk slides copy <presentationId> <title>`
- `wk slides export <presentationId> --format pdf|pptx`

## Slide-level CRUD
- `wk slides list-slides <presentationId>`
- `wk slides read-slide <presentationId> <slideId>`
- `wk slides add-slide <presentationId> <image> [--notes ...]`
- `wk slides replace-slide <presentationId> <slideId> <image> [--notes ...]`
- `wk slides update-notes <presentationId> <slideId> --notes "..."`
- `wk slides delete-slide <presentationId> <slideId>`

## Batch operations
- Treat repeated slide edits as scripted loops over `list-slides` IDs.

## Template creation pattern
- Maintain a template deck and clone it with `wk slides copy ...`, then mutate target slides.

## Example
```bash
wk slides create-from-markdown 'QBR 2026-02' --content-file ./qbr.md --dry-run
wk slides update-notes <presentationId> <slideId> --notes 'Talk track v2' --dry-run
```
