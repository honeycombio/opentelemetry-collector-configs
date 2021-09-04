set -e

read -p 'What version should we be releasing? v' VERSION
make release
git tag v$VERSION
git push origin v$VERSION
gh release create v$VERSION ./dist/*