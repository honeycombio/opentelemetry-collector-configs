# Release Process

1. Ensure you have these dependencies in your `$PATH`:
   - [go](https://golang.org/dl/)
   - [jq](https://stedolan.github.io/jq/download/)
   - [yq](https://kislyuk.github.io/yq/#installation)
   - [ocb](https://github.com/open-telemetry/opentelemetry-collector/releases) - you must install the version that matches the `builder-config.yaml` version.
   - [docker](https://docs.docker.com/get-docker/)
   - [fpm](https://fpm.readthedocs.io/en/v1.13.1/installing.html)
   - [gh](https://github.com/cli/cli#installation)
2. Add release entry to `timestampprocessor/CHANGELOG.md` 
3. Update timestampprocessor entry in the example in `docs/building.md` with the new release version
4. Open a PR with the above, and merge that into main
5. Get the latest from main.
6. Run ./release.sh and follow prompts.  This will push a tag and create a release.
7. Update release notes with changelog entry 
8. Manually create and then push tag `timestampprocessor/v{new version number}`