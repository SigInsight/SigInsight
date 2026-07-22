Follow these steps if you want to monitor system metrics like Total Requests, Total Ingress / Egress, and Total Errors etc., of your Azure Blob Storage.

&nbsp;

## Prerequisites

- Azure Subscription and Azure Blob storage instance running
- Central Collector Setup

&nbsp;

## Query Metrics

Once you have completed the prerequisites, you can start monitoring your Azure Blob Storage's system metrics with SigInsight.

1. Log in to your SigInsight account.
2. Open Metrics Explorer.
3. Select `azure_ingress_total` and use **Avg By** with the `location` tag.
4. Filter with `name = <storage-account-name>`.


That's it! You have successfully set up monitoring for your Azure Blob Storage's system metrics with SigInsight.

&nbsp;

If you encounter any difficulties, please refer to this [troubleshooting section](https://signoz.io/docs/azure-monitoring/az-blob-storage/metrics/#troubleshooting)
