#!/usr/bin/env python3
"""
Strip ALL inline class attributes from Vue template sections.

Prepares the project for Flowbite Vue migration by removing all
Tailwind utility classes from templates, leaving bare HTML structure.

Usage:
    python scripts/strip_inline_styles.py [--dry-run]
"""

import re
import sys
from pathlib import Path


def strip_template_classes(content: str) -> tuple[str, int]:
    """
    Remove all class="..." and :class="..." attributes from the <template> section.
    Leaves <script> and <style> blocks untouched.
    """
    # Split into template / script+style sections
    # We only want to touch content inside <template>...</template>
    template_match = re.search(
        r'(<template>)(.*?)(</template>)',
        content,
        flags=re.DOTALL,
    )
    if not template_match:
        return content, 0

    before = content[:template_match.start()]
    template_open = template_match.group(1)
    template_body = template_match.group(2)
    template_close = template_match.group(3)
    after = content[template_match.end():]

    changes = 0

    # 1. Remove static class="..." (single or double quoted, possibly multiline)
    def remove_static_class(m):
        nonlocal changes
        changes += 1
        return ''

    template_body = re.sub(
        r'''\s+class=(?:"[^"]*"|'[^']*')''',
        remove_static_class,
        template_body,
        flags=re.DOTALL,
    )

    # 2. Remove dynamic :class="..." (single or double quoted, possibly multiline)
    template_body = re.sub(
        r'''\s+:class=(?:"[^"]*"|'[^']*')''',
        remove_static_class,
        template_body,
        flags=re.DOTALL,
    )

    new_content = before + template_open + template_body + template_close + after
    return new_content, changes


def main():
    dry_run = '--dry-run' in sys.argv

    frontend_src = Path('frontend/src')
    if not frontend_src.exists():
        print(f'Error: {frontend_src} not found. Run from project root.', file=sys.stderr)
        sys.exit(1)

    vue_files = sorted(frontend_src.rglob('*.vue'))
    print(f'Found {len(vue_files)} Vue files\n')

    total_changes = 0
    modified = []

    for path in vue_files:
        original = path.read_text(encoding='utf-8')
        cleaned, n = strip_template_classes(original)

        if n == 0:
            continue

        modified.append((path, n))
        total_changes += n

        rel = path.relative_to(frontend_src)
        print(f'  {"[dry]" if dry_run else "✓"} {rel}  ({n} class attrs removed)')

        if not dry_run:
            path.write_text(cleaned, encoding='utf-8')

    print(f'\n{"─"*50}')
    print(f'Files modified : {len(modified)}')
    print(f'Attrs removed  : {total_changes}')
    if dry_run:
        print('(dry run — no files written)')


if __name__ == '__main__':
    main()
