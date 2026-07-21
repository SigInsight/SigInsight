export const PANEL_QUERY_CACHE_TIME = 30_000;
// keep it low or zero, otherwise, when enabled auto-refresh, this causes OOM due to accumulated queries in cache
export const PANEL_QUERY_CACHE_TIME_ON_AUTO_REFRESH = 0;
