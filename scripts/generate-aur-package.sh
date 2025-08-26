#!/bin/bash

# Script to generate AUR package
# Usage: ./scripts/generate-aur-package.sh <tag> <owner> <repo>

set -euo pipefail

if [ $# -ne 3 ]; then
    echo "Usage: $0 <tag> <owner> <repo>"
    echo "Example: $0 v0.3.8 joelgrimberg projector"
    exit 1
fi

TAG=$1
OWNER=$2
REPO=$3

echo "Generating AUR package for $TAG..."

# Create AUR directory
mkdir -p dist/aur

# Download source tarball to compute SHA256
echo "Downloading source tarball for SHA256 computation..."
curl -fSL -o "dist/aur/projector-$TAG.tar.gz" "https://github.com/$OWNER/$REPO/archive/$TAG.tar.gz"

# Compute SHA256 for source tarball
SOURCE_SHA256=$(shasum -a 256 "dist/aur/projector-$TAG.tar.gz" | cut -d' ' -f1)
echo "Source tarball SHA256: $SOURCE_SHA256"

# Extract version without 'v' prefix
PKGVER=${TAG#v}

# Generate PKGBUILD with correct version and SHA256
cat > dist/aur/PKGBUILD << EOF
# Maintainer: Joël Grimberg <joelgrimberg@users.noreply.github.com>
pkgname=projector-cli
pkgver=$PKGVER
pkgrel=1
pkgdesc="A CLI application for project and action management"
arch=('x86_64' 'aarch64')
url="https://github.com/joelgrimberg/projector"
license=('MIT')
depends=('glibc')
makedepends=('go' 'git')
source=("projector-\${pkgver}.tar.gz::https://github.com/joelgrimberg/projector/archive/$TAG.tar.gz")
sha256sums=('$SOURCE_SHA256')

build() {
  cd "projector-\${pkgver}"
  go build -ldflags="-s -w" -o "\$pkgname" .
}

package() {
  cd "projector-\${pkgver}"
  
  # Install binary
  install -Dm755 "\$pkgname" "\$pkgdir/usr/bin/\$pkgname"
  
  # Install license
  install -Dm644 LICENSE "\$pkgdir/usr/share/licenses/\$pkgname/LICENSE"
  
  # Install README
  install -Dm644 README.md "\$pkgdir/usr/share/doc/\$pkgname/README.md"
}
EOF

echo "AUR PKGBUILD generated successfully:"
cat dist/aur/PKGBUILD

# Create AUR package tarball
cd dist/aur
tar -czf "projector-cli-$TAG.tar.gz" PKGBUILD
cd ../..

echo "AUR package tarball created: dist/aur/projector-cli-$TAG.tar.gz"
echo "Ready for AUR submission!"
