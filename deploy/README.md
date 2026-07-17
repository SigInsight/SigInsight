# Deploy

Check that you have cloned [SigInsight/SigInsight](https://github.com/SigInsight/SigInsight)
and currently are in `SigInsight/deploy` folder.

## Docker

If you don't have docker set up, please follow [this guide](https://docs.docker.com/engine/install/)
to set up docker before proceeding with the next steps.

### Using Install Script

Now run the following command to install:

```sh
./install.sh
```

### Using Docker Compose

If you don't have docker compose set up, please follow [this guide](https://docs.docker.com/compose/install/)
to set up docker compose before proceeding with the next steps.

```sh
cd deploy/docker
docker compose up -d
```

Open http://localhost:8080 in your favourite browser.

By default, the compose files use these GHCR images:

```sh
ghcr.io/siginsight/siginsight:v1.4.0
ghcr.io/siginsight/signoz-otel-collector:v1.0.0
ghcr.io/siginsight/clickhouse-init-histogram-quantile:25.5.6-latest
```

The `v1.0.0` collector image currently supports Linux AMD64 hosts only.

You can override their tags with environment variables:

```sh
export VERSION=v1.4.0
export OTELCOL_TAG=v1.0.0
export HISTOGRAM_QUANTILE_INIT_IMAGE=ghcr.io/siginsight/clickhouse-init-histogram-quantile:25.5.6-latest
```

To use a different SigInsight registry or repository, set the complete image reference:

```sh
export SIGNOZ_IMAGE=ghcr.io/siginsight/siginsight:v1.4.0
```

To start collecting logs and metrics from your infrastructure, run the following command:

```sh
cd generator/infra
docker compose up -d
```

To start generating sample traces, run the following command:

```sh
cd generator/hotrod
docker compose up -d
```

In a couple of minutes, you should see the data generated from hotrod in the SigInsight UI.

For more details, please refer to the repository-level deployment instructions and compose files in this repo.

## Docker Swarm

To install SigInsight using Docker Swarm, run the following command:

```sh
cd deploy/docker-swarm
docker stack deploy -c docker-compose.yaml siginsight
```

Open http://localhost:8080 in your favourite browser.

To start collecting logs and metrics from your infrastructure, run the following command:

```sh
cd generator/infra
docker stack deploy -c docker-compose.yaml infra
```

To start generating sample traces, run the following command:

```sh
cd generator/hotrod
docker stack deploy -c docker-compose.yaml hotrod
```

In a couple of minutes, you should see the data generated from hotrod in the SigInsight UI.

For more details, please refer to the compose files in this repository.

## Uninstall/Troubleshoot?

Go to the repository root README for the latest SigInsight deployment notes.

