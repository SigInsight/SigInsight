export function isLightweightQueryEditorEnabled(): boolean {
	return process.env.LIGHTWEIGHT_QUERY_EDITOR_ENABLED === 'true';
}
