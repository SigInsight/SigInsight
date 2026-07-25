export const explorerView = {
	status: 'success',
	data: [
		{
			id: 'test-uuid-1',
			name: 'Table View',
			category: '',
			createdAt: '2023-08-29T18:04:10.906310033Z',
			createdBy: 'test-user-1',
			updatedAt: '2024-01-29T10:42:47.346331133Z',
			updatedBy: 'test-user-1',
			sourcePage: 'traces',
			tags: [''],
			compositeQuery: {
				panelType: 'table',
				queryType: 'builder',
				unit: undefined,
				queries: [
					{
						type: 'builder_query',
						spec: {
							name: 'A',
							signal: 'traces',
							stepInterval: 60,
							filter: { expression: "component != 'test-component'" },
							groupBy: [
								{
									name: 'component',
									fieldDataType: 'string',
									fieldContext: 'attribute',
								},
								{
									name: 'client-uuid',
									fieldDataType: 'string',
									fieldContext: 'resource',
								},
							],
							order: [{ key: { name: 'timestamp' }, direction: 'desc' }],
							aggregations: [{ expression: 'count()' }],
							disabled: false,
						},
					},
				],
			},
			extraData: '{"color":"#00ffd0"}',
		},
		{
			id: '8c4bf492-d54d-4ab2-a8d6-9c1563f46e1f',
			name: 'R-test panel',
			category: '',
			createdAt: '2024-07-01T13:45:57.924686766Z',
			createdBy: 'test-user-test',
			updatedAt: '2024-07-01T13:48:31.032106578Z',
			updatedBy: 'test-user-test',
			sourcePage: 'traces',
			tags: [''],
			compositeQuery: {
				panelType: 'list',
				queryType: 'builder',
				unit: undefined,
				queries: [
					{
						type: 'builder_query',
						spec: {
							name: 'A',
							signal: 'traces',
							stepInterval: 60,
							filter: { expression: "httpMethod = 'GET'" },
							order: [{ key: { name: 'timestamp' }, direction: 'desc' }],
							aggregations: [{ expression: 'count()' }],
							disabled: false,
						},
					},
				],
			},
			extraData: '{"color":"#AD7F58"}',
		},
	],
};
