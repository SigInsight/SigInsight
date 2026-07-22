## Prerequisite

- Azure subscription and Database instance running
- Central Collector Setup
- [SQL monitoring profile](https://learn.microsoft.com/en-us/azure/azure-sql/database/sql-insights-enable?view=azuresql#create-sql-monitoring-profile) created to monitor the databases in Azure Monitor

&nbsp;


## Query Metrics

Once you have completed the prerequisites, you can start monitoring your Database's system metrics with SigInsight. Here's how you can do it:

1. Log in to your SigInsight account.
2. Open Metrics Explorer.
3. Select `azure_storage_maximum` and use **Avg By** with the `location` tag.
4. Filter with `name = <database-name>`.

That's it! You have successfully set up monitoring for your Database's system metrics with SigInsight.

&nbsp;

**NOTE:**
Make sure you have created a sql monitoring profile in Azure Monitor if not, follow this guide to [Create SQL Monitoring Profile](https://learn.microsoft.com/en-us/azure/azure-sql/database/sql-insights-enable?view=azuresql#create-sql-monitoring-profile).
You can monitor multiple databases in a single profile.

&nbsp;

If you encounter any difficulties, please refer to this [troubleshooting section](https://signoz.io/docs/azure-monitoring/db-metrics/#troubleshooting)
