#!/usr/bin/env bash
# Stage the marketing website static assets for Docker / release packaging.
# The website is a pure static site (no bundler), so "build" = staging the
# publishable files into website/dist, mirroring scripts/build_admin_dist.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WEBSITE_DIR="$PROJECT_ROOT/website"
DIST_DIR="$WEBSITE_DIR/dist"

cd "$WEBSITE_DIR"

# Files that ship to production
PUBLISH_FILES=(
	index.html
	product.html
	scenes.html
	service-members.html
	pricing.html
	security.html
	privacy.html
	terms.html
	404.html
	styles.css
	main.js
	sitemap.xml
	robots.txt
)

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR/assets"

for f in "${PUBLISH_FILES[@]}"; do
	if [ ! -f "$f" ]; then
		echo "error: expected file missing: website/$f" >&2
		exit 1
	fi
	cp "$f" "$DIST_DIR/"
done

# Binary assets (logo variants, product screenshots)
find assets -maxdepth 1 -type f -exec cp {} "$DIST_DIR/assets/" \;

echo ">> website dist staged at website/dist"
find "$DIST_DIR" -type f | sort | sed "s|$DIST_DIR/||"
