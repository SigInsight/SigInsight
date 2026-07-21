Follow these steps if you want to monitor System metrics like CPU Percentage, Memory Percentage etc. of your Azure Container App.

&nbsp;

## Prerequisites

- Azure subscription and an Azure Container App instance running
- Central Collector Setup

&nbsp;

# Query Metrics

Once you have completed the prerequisites, you can start monitoring your Azure Container App's system metrics with SigNoz. Here's how you can do it:

1. Log in to your SigNoz account.
2. Open Metrics Explorer.
3. Select `azure_replicas_count` and use **Avg By** with the `name` tag.
4. Filter with `type = Microsoft.App/containerApps`.

This query lets you monitor system metrics of your Azure Container App in SigNoz.

&nbsp;

If you encounter any difficulties, please refer to this [troubleshooting section](https://signoz.io/docs/azure-monitoring/az-container-apps/metrics/#troubleshooting)
