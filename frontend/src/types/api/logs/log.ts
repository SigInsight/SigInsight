export interface ILog {
	date: string;
	timestamp: number | string;
	id: string;
	span_id?: string;
	trace_id?: string;
	trace_flags: number;
	severity_text: string;
	severity_number: number;
	body: string;
	resources_string: Record<string, never>;
	scope_string: Record<string, never>;
	attributes_string: Record<string, never>;
}

type OmitAttributesResources = Pick<
	ILog,
	Exclude<keyof ILog, 'resources_string' | 'scope_string' | 'attributes_string'>
>;

export type ILogAggregateAttributesResources = OmitAttributesResources & {
	attributes: Record<string, never>;
	resources: Record<string, never>;
	scope: Record<string, never>;
};
