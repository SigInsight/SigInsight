export const quickFiltersListResponse = {
	status: 'success',
	data: {
		signal: 'logs',
		filters: [
			{
				key: 'os.description',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'service.name',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'duration_nano',
				dataType: 'float64',
				type: 'tag',
			},
			{
				key: 'quantity',
				dataType: 'float64',
				type: 'tag',
			},
			{
				key: 'body',
				dataType: 'string',
				type: '',
			},
			{
				key: 'service.namespace',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'cloud.region',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'service.instance.id',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'host.arch',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'process.owner',
				dataType: 'string',
				type: 'resource',
			},
		],
	},
};

export const otherFiltersResponse = {
	status: 'success',
	data: {
		attributes: [
			{
				key: 'service.name',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'cloud.availability_zone',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'service.namespace',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'cloud.region',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'service.instance.id',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'host.arch',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'process.pid',
				dataType: 'string',
				type: 'resource',
			},
			{
				key: 'os.description',
				dataType: 'string',
				type: 'resource',
			},
		],
	},
};

export const quickFiltersAttributeValuesResponse = {
	status: 'success',
	data: {
		stringAttributeValues: [
			'mq-kafka',
			'otel-demo',
			'otlp-python',
			'sample-flask',
		],
		numberAttributeValues: null,
		boolAttributeValues: null,
	},
};
