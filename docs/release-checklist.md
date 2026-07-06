# SigInsight GitHub Launch Checklist

## Repository

- Confirm the canonical repository is `https://github.com/SigInsight/SigInsight`.
- Confirm the default branch is `main`.
- Confirm branch protection is enabled for `main`.
- Confirm required GitHub Actions checks match the workflows that still exist in `.github/workflows`.
- Confirm `README.md` and `README.zh-cn.md` are the only top-level README variants linked from the repository landing page.

## Container Images

- Confirm the application image publishes to `ghcr.io/siginsight/siginsight`.
- Confirm the ClickHouse helper image publishes to `ghcr.io/siginsight/clickhouse-init-histogram-quantile`.
- Confirm package visibility and pull permissions in GHCR are correct for your intended deployment model.
- Confirm `latest`, `main-<sha>`, and release tags are pushed as expected.
- Confirm a fresh `docker pull ghcr.io/siginsight/siginsight:latest` works from a clean environment.

## GitHub Actions

- Confirm `build-community.yaml` runs successfully on a push to `main`.
- Confirm `gor-histogramquantile.yaml` runs successfully and publishes the helper image.
- Confirm `goci.yaml`, `jsci.yaml`, and `commitci.yaml` still match the current repository structure.
- Confirm deprecated Python integration paths remain disabled and clearly marked as deprecated.
- Confirm there are no required status checks pointing to deleted workflows.

## Deployment

- Confirm `deploy/docker/docker-compose.yaml` starts successfully with the default GHCR image values.
- Confirm `deploy/docker/docker-compose.ha.yaml` starts successfully with the default GHCR image values.
- Confirm `deploy/docker-swarm/docker-compose.yaml` references the GHCR images and deploys successfully.
- Confirm `deploy/docker-swarm/docker-compose.ha.yaml` references the GHCR images and deploys successfully.
- Confirm the `init-clickhouse` service copies `histogramQuantile` from `/opt/siginsight/bin/histogramQuantile`.

## Documentation

- Confirm root deployment docs point users to GHCR-based images.
- Confirm clone instructions reference `SigInsight/SigInsight`.
- Confirm discussions, contributors, and repository links point to `SigInsight/SigInsight`.
- Confirm no top-level docs still link to the deleted German or Portuguese README files.
- Confirm any future external docs or website pages are updated before public announcement.

## Release Validation

- Confirm a production image built from the current `main` branch returns a healthy HTTP response after startup.
- Confirm the frontend assets are included in the image and the UI loads normally.
- Confirm the login flow works without the old license-gating regression.
- Confirm the ClickHouse helper image version used by deployments matches the published GHCR tag.
- Confirm at least one end-to-end smoke deployment is run after the first GitHub release.