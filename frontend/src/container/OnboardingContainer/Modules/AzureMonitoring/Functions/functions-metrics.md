Follow these steps if you want to monitor System metrics like CPU Percentage, Memory Percentage etc. of your Azure Functions.

&nbsp;

## Prerequisites

- Azure subscription and an Azure Container App instance running
- Central Collector Setup

&nbsp;

## Query Metrics

Once you have completed the prerequisites, you can start monitoring your Azure Function's system metrics with SigNoz. Here's how you can do it:

1. Log in to your SigNoz account.
2. Open Metrics Explorer.
3. Select `azure_requests_total` and use **Avg By** with the `location` tag.
4. Filter with `name = <function-name>`.


That's it! You have successfully set up monitoring for your Azure Function's system metrics with SigNoz.

&nbsp;

If you encounter any difficulties, please refer to this [troubleshooting section](https://signoz.io/docs/azure-monitoring/az-fns/metrics/#troubleshooting)
