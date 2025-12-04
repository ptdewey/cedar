# Cedar

A lightweight static site generator built in Go with native ATProto publishing support.

Cedar takes markdown content, renders it through Go templates, and outputs a static site. It can also publish your content to the AT Protocol (Bluesky/ATProto) as `site.standard.document` records, making your posts discoverable across the ATProto network.

## Installation

```sh
go install github.com/ptdewey/cedar@latest
```

From source:

```sh
git clone https://github.com/ptdewey/cedar.git
cd cedar
go install .
```

Nix:

```sh
nix build
# or
nix run . -- build
```

## Quick start

Create a project directory with the following structure:

```
my-site/
  cedar.toml
  content/
    index.md
    posts/
      hello-world.md
  templates/
    base.tmpl
    index.tmpl
    post.tmpl
    partials/
  static/
```

See the `example/` directory for a working reference.

### Build the site

```sh
cedar build
```

This reads `cedar.toml`, processes all markdown files, renders templates, and writes the result to `public/`.

## Configuration

Cedar uses a TOML config file (also supports JSON and YAML). Pass a custom path with `--config`:

```sh
cedar build --config my-config.toml
```

Full example:

```toml
content_dir = "content"
template_dir = "templates"
template_ext = ".tmpl"
publish_dir = "public"
static_dir = "static"
base_template_path = "base.tmpl"
clean_build = false
build_draft = false
build_future = false
allow_unsafe_html = false

[atproto]
handle = "yourhandle.bsky.social"

[atproto.publications.blog]
name = "My Blog"
url = "https://example.com"
description = "A blog about things"

[rss]
generate = true
title = "My Blog"
description = "A blog about things"
url = "https://example.com"

[[routes]]
content_path = "index.md"
output_pattern = "/"
template = "index.tmpl"

[[routes]]
content_path = "posts"
output_pattern = "/posts/:slug"
template = "post.tmpl"
generate_rss = true
publish = "blog"
```

### Top-level fields

| Field              | Default     | Description                                      |
| ------------------ | ----------- | ------------------------------------------------ |
| content_dir        | `content`   | Directory containing markdown files              |
| template_dir       | `templates` | Directory containing Go templates                |
| template_ext       | `.tmpl`     | File extension for templates                     |
| publish_dir        | `public`    | Output directory for the built site              |
| static_dir         | `static`    | Static files copied as-is to publish_dir         |
| base_template_path | (none)      | Path to base template relative to template_dir   |
| cache_dir          | `build`     | Temporary build directory (removed after build)  |
| clean_build        | `false`     | Remove publish_dir before building               |
| build_draft        | `false`     | Include pages with `draft: true` in front matter |
| build_future       | `false`     | Include pages with a future date                 |
| allow_unsafe_html  | `false`     | Allow raw HTML in markdown                       |
| copyright          | (none)      | Copyright string available in templates          |

### Routes

Each route maps a content path to an output URL pattern and template:

```toml
[[routes]]
content_path = "posts"          # directory under content_dir, or a single file
output_pattern = "/posts/:slug" # :slug is replaced by the filename without extension
template = "post.tmpl"          # template file in template_dir
generate_rss = true             # include this route in the RSS feed
publish = "blog"                # publish to the "blog" publication (see ATProto section)
```

For a single-file route like the index page:

```toml
[[routes]]
content_path = "index.md"
output_pattern = "/"
template = "index.tmpl"
```

Only routes with a `publish` value matching a configured publication key are synced to ATProto. Routes are opt-out by default (empty string = don't publish).

## Content

Content files are markdown with YAML front matter:

```markdown
---
title: Hello World
date: 2026-02-21
description: A short description for metadata.
tags:
  - go
  - web
draft: false
---

Your markdown content here.
```

The `title`, `date`, `description`, and `tags` fields are used by ATProto publishing. All front matter fields are available in templates via `.Metadata`.

Markdown is parsed with goldmark and supports GFM (tables, strikethrough, autolinks, task lists) and footnotes. Code blocks get syntax highlighting via chroma.

## Templates

Cedar uses Go's `html/template` package. Templates receive an `htmlPage` struct with these fields:

| Field          | Type                  | Description                                                      |
| -------------- | --------------------- | ---------------------------------------------------------------- |
| .Metadata      | map[string]any        | Front matter from the markdown file                              |
| .HTMLContent   | template.HTML         | Rendered HTML from the markdown body                             |
| .AllPages      | map[string][]PageInfo | All pages grouped by route content_path                          |
| .ATProtoDocURI | string                | AT-URI for this page's published document (empty if unpublished) |
| .ATProtoDID    | string                | Resolved DID for the configured ATProto handle                   |
| .ATProtoPDS    | string                | PDS service endpoint URL for the resolved DID                    |

Each `PageInfo` in `.AllPages` has:

| Field     | Type           | Description                |
| --------- | -------------- | -------------------------- |
| .Metadata | map[string]any | Front matter               |
| .Slug     | string         | Filename without extension |
| .Date     | time.Time      | Parsed date                |

### Template functions

| Function                  | Description                                                              |
| ------------------------- | ------------------------------------------------------------------------ |
| `getStr .Metadata "key"`  | Safely get a string from a metadata map                                  |
| `linkItem "/url" "Title"` | Create a LinkItem struct with .Link and .Title fields                    |
| `writingItems .pages`     | Convert a []PageInfo into []WritingItem with .Date, .Link, .Title, .Type |
| `formatDate "2026-02-21"` | Format a date string as "Feb 21, 2026"                                   |

### Example templates

Base template (`base.tmpl`):

```gotmpl
{{define "base"}}
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>{{template "title" .}}</title>
  </head>
  <body>
    <nav><a href="/">Home</a></nav>
    {{template "content" .}}
  </body>
</html>
{{end}}
```

Page template (`post.tmpl`):

```gotmpl
{{template "base" .}}
{{define "title"}} {{.Metadata.title}} {{end}}
{{define "content"}}
<article>
  <h1>{{.Metadata.title}}</h1>
  <p>{{getStr .Metadata "date"}}</p>
  {{.HTMLContent}}
</article>
{{end}}
```

Index with page listing (`index.tmpl`):

```gotmpl
{{template "base" .}}
{{define "title"}}Home{{end}}
{{define "content"}}
{{.HTMLContent}}
<ul>
  {{range index .AllPages "posts"}}
  <li><a href="/posts/{{.Slug}}">{{getStr .Metadata "title"}}</a></li>
  {{end}}
</ul>
{{end}}
```

Partial templates placed in `templates/partials/` are automatically loaded and available via `{{template "name" .}}`.

## ATProto publishing

Cedar can publish your content to the AT Protocol as `site.standard.document` records, making posts available on Bluesky and other ATProto-compatible readers.

### Publications

Cedar supports multiple named publications under a single ATProto handle. Each publication becomes a separate `site.standard.publication` record with its own name, URL, and settings. Routes reference publications by key:

```toml
[atproto]
handle = "yourhandle.bsky.social"

[atproto.publications.blog]
name = "My Blog"
url = "https://example.com"
description = "Long-form posts"

[atproto.publications.notes]
name = "Quick Notes"
url = "https://example.com/notes"
description = "Short thoughts"
publish_leaflet = true

[[routes]]
content_path = "posts"
output_pattern = "/posts/:slug"
template = "post.tmpl"
publish = "blog"

[[routes]]
content_path = "notes"
output_pattern = "/notes/:slug"
template = "note.tmpl"
publish = "notes"
```

Each publication has these fields:

| Field           | Description                                                |
| --------------- | ---------------------------------------------------------- |
| name            | Display name for the publication                           |
| url             | URL of the site/viewer for this publication                |
| description     | Short description (optional)                               |
| publish_leaflet | Also publish `pub.leaflet.content` records (default false) |

### Setup

1. Add the `[atproto]` section with your handle and at least one publication.
2. Set `publish = "<publication-key>"` on routes you want to publish.
3. Authenticate:

```sh
cedar auth
```

This opens an OAuth flow in your browser and stores credentials locally.

4. Publish:

```sh
cedar publish
```

Cedar tracks content hashes and only creates/updates records that have changed.

5. Rebuild to generate the verification endpoint:

```sh
cedar build
```

This writes `.well-known/site.standard.publication` to your publish directory, which links your domain to your ATProto publications.

### What gets published

For each configured publication, Cedar creates a `site.standard.publication` record on first publish. For each page in a route that references that publication, Cedar creates a `site.standard.document` record containing:

- Title, description, tags from front matter
- The raw markdown wrapped as `at.markpub.markdown` with GFM flavor
- A plain text extraction for search indexing
- The page's URL path on your site

### Dry run

Inspect the records that would be published without authenticating:

```sh
cedar publish --dry-run
```

### Leaflet block publishing

Publications with `publish_leaflet = true` also get `pub.leaflet.content` records alongside their documents. This converts your markdown into Leaflet's native block format (text blocks with byte-offset rich text facets, code blocks, headers, lists, blockquotes, etc.) so your content renders natively in the Leaflet viewer.

To preview how your content would look as leaflet blocks without publishing:

```sh
cedar publish --preview
```

This generates HTML files in `_preview/` that approximate Leaflet's rendering.

### Fetching Leaflet records client-side

If you write posts in Leaflet's editor and want them to appear alongside your static site content, you can fetch `pub.leaflet.content` records from your PDS directly in the browser. The DID and PDS URL are available in templates:

```gotmpl
{{if .ATProtoDID}}
<script>
  (async () => {
    const did = "{{.ATProtoDID}}";
    const pds = "{{.ATProtoPDS}}";
    const res = await fetch(
      `${pds}/xrpc/com.atproto.repo.listRecords?repo=${did}&collection=pub.leaflet.content`,
    );
    const data = await res.json();
    const list = document.getElementById("leaflet-posts");
    for (const rec of data.records || []) {
      const rkey = rec.uri.split("/").pop();
      const li = document.createElement("li");
      const a = document.createElement("a");
      a.href = `https://leaflet.pub/lish/${did}/default/${rkey}`;
      a.textContent = rkey;
      li.appendChild(a);
      list.appendChild(li);
    }
  })();
</script>
<ul id="leaflet-posts"></ul>
{{end}}
```

## RSS

Enable RSS generation in the config:

```toml
[rss]
generate = true
title = "My Blog"
description = "A blog about things"
url = "https://example.com"
```

Then set `generate_rss = true` on routes to include in the feed. The RSS file is written to `publish_dir/rss.xml`.

## Commands

```
cedar build     Build the static site
cedar auth      Authenticate with ATProto via OAuth
cedar publish   Publish/sync content to ATProto PDS
```

All commands accept `--config <path>` (default: `cedar.toml`).

Publish flags:

- `--dry-run` — print records as JSON without publishing
- `--preview` — generate leaflet HTML preview files in `_preview/`

## License

MIT
