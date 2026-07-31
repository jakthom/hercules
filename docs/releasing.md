# Releasing

Hercules releases are driven by an annotated `v*` tag. The tag without its `v` prefix must exactly match `.VERSION`.

## Preparation

1. Move the relevant `CHANGELOG.md` section from Unreleased to the release date.
2. Set `.VERSION` to the semantic version without `v`.
3. Run `make quality`, `make build`, and `make check` on a clean tree.
4. Confirm `go mod tidy` produces no diff.
5. Review dependency and container updates and the current vulnerability scan.
6. Merge through the normal protected branch process and wait for CI.

## Publish

```sh
git tag -s v0.9.0 -m "Hercules v0.9.0"
git push origin v0.9.0
```

The release workflow verifies the tag/version match, reruns GoReleaser checks/tests, cross-compiles CGO Linux amd64 and arm64 binaries, creates tarballs and SHA-256 checksums, publishes the GitHub release, and publishes an amd64/arm64 non-root image to GitHub Container Registry with provenance and an SBOM.

If signing keys are unavailable, use an annotated tag only after explicitly documenting that exception. Never retag a published version; increment the version and issue another release.

## Verification

Download one archive and verify it with `checksums.txt`, run `hercules version`, run a full `hercules check` against a known configuration, pull both container architectures, and confirm the GitHub release notes and image tags. Mark a release complete only after these checks pass.
