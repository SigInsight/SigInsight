Follow these steps if you want to monitor System metrics like CPU Percentage, Memory Percentage etc. of your Azure App Service.

&nbsp;

## Prerequisites

- EventHub Setup
- Central Collector Setup

## Query Metrics

Once you have completed the prerequisites, you can start monitoring your Azure App Service's system metrics with SigNoz Cloud. Here's how you can do it:

1. Log in to your SigNoz account.
2. Open Metrics Explorer.
3. Select `azure_memorypercentage_total` and use **Avg By** with the `location` tag.
4. Filter with `name = <app-svc-plan-name>`.

This query shows the memory usage of your Azure App Service in SigNoz Cloud.

&nbsp;

If you encounter any difficulties, please refer to this [troubleshooting section](https://signoz.io/docs/azure-monitoring/app-service/metrics/#troubleshooting)
