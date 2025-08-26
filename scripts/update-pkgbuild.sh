#!/bin/bash

# Script to update PKGBUILD version
# Usage: ./scripts/update-pkgbuild.sh <new_version>

if [ $# -eq 0 ]; then
    echo "Usage: $0 <new_version>"
    echo "Example: $0 0.3.8"
    exit 1
fi

NEW_VERSION=$1
PKGBUILD_FILE="PKGBUILD"

if [ ! -f "$PKGBUILD_FILE" ]; then
    echo "Error: PKGBUILD file not found"
    exit 1
fi

# Update version in PKGBUILD
sed -i.bak "s/^pkgver=.*/pkgver=$NEW_VERSION/" "$PKGBUILD_FILE"

# Remove backup file
rm -f "${PKGBUILD_FILE}.bak"

echo "Updated PKGBUILD version to $NEW_VERSION"
echo "Don't forget to:"
echo "1. Compute SHA256: sha256sum projector-$NEW_VERSION.tar.gz"
echo "2. Update sha256sums in PKGBUILD"
echo "3. Test build: makepkg -f"
echo "4. Commit changes: git add PKGBUILD && git commit -m 'chore: update PKGBUILD to v$NEW_VERSION'"
